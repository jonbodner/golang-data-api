package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"

	"github.com/google/uuid"

	"github.com/go-playground/validator/v10"
)

type data struct {
	ID      string `json:"ID" validate:"required"`
	Message string `json:"Message" validate:"required"`
}

func (d data) String() string {
	return fmt.Sprintf("ID:%s,Message:%s", d.ID, d.Message)
}

var emptyData = data{}

type allData []data

func (a allData) String() string {
	returnData := ""

	for _, x := range serviceData {
		returnData += "{" + x.String() + "}"
	}

	return "[" + returnData + "]"
}

var serviceData allData

type info struct {
	NAME string `json:"service-name"`
	ID   string `json:"service-id"`
}

func (i info) String() string {
	out, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(out)
}

var serviceInfo = info{}

var incorrectInputError string = "Please check submission: {\"ID\":\"<ID_VALUE>\",\"Message\":\"<MESSAGE_VALUE>\"}"

var serviceId uuid.UUID = uuid.New()

var standardFields log.Fields

var validate *validator.Validate

func logErrors(event string, message string, errorData ...string) {
	_, fileName, line, _ := runtime.Caller(1)
	log.WithFields(log.Fields{"event": event, "line": line, "file": fileName, "data": errorData}).Error(message)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func getServiceInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, serviceInfo.String())
}

func searchData(dataID string) (foundData data) {
	for _, x := range serviceData {
		if x.ID == dataID {
			foundData = x
		}
	}

	return
}

func createData(w http.ResponseWriter, r *http.Request) {
	var newData data
	input, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logErrors("read-all", incorrectInputError)
		fmt.Fprint(w, incorrectInputError)
		return
	}

	if !json.Valid(input) {
		logErrors("json-not-welformed", "", string(input))
		fmt.Fprint(w, incorrectInputError)
		return
	}

	err = json.Unmarshal(input, &newData)
	if err != nil {
		logErrors("unmarshal", "", newData.String())
		fmt.Fprint(w, incorrectInputError)
		return
	}

	err = validate.Struct(newData)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			logErrors("validate-errors", "", err.Error())
		}

		for _, err := range err.(validator.ValidationErrors) {
			logErrors("validate-errors", fmt.Sprintf("[%s,%s]", err.Field(), err.Tag()))
		}

		logErrors("invalid-data-struct", "", newData.String())
		fmt.Fprint(w, incorrectInputError)

		return
	}

	foundData := searchData(newData.ID)

	if foundData == emptyData {
		serviceData = append(serviceData, newData)

		err = json.NewEncoder(w).Encode(newData)
		if err != nil {
			logErrors("encode", "JSON Encoding in createData", newData.String())
			return
		}
	} else {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "ERROR: Data exists.")
	}
}

func getData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]

	foundData := searchData(dataID)

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(foundData)
	if err != nil {
		logErrors("encode", "JSON Encoding in getData", foundData.String())
	}
}

func getAllData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(serviceData)
	if err != nil {
		logErrors("unmarshal", "", serviceData.String())
	}
}

func updateData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]
	var updatedData data

	input, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logErrors("read", incorrectInputError)
		fmt.Fprint(w, incorrectInputError)
		return
	}

	if !json.Valid(input) {
		logErrors("json-not-welformed", "", string(input))
		fmt.Fprint(w, incorrectInputError)
		return
	}

	err = json.Unmarshal(input, &updatedData)
	if err != nil {
		logErrors("unmarshal", "", updatedData.String())
		fmt.Fprint(w, incorrectInputError)
		return
	}

	err = validate.Struct(updatedData)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			logErrors("validate-errors", "", err.Error())
		}

		for _, err := range err.(validator.ValidationErrors) {
			logErrors("validate-errors", fmt.Sprintf("[%s,%s]", err.Field(), err.Tag()))
		}

		logErrors("invalid-data-struct", "", updatedData.String())
		fmt.Fprint(w, incorrectInputError)

		return
	}

	for i, singleData := range serviceData {
		if singleData.ID == dataID {
			if singleData != updatedData {
				singleData.Message = updatedData.Message
				serviceData = append(serviceData[:i], singleData)
				err = json.NewEncoder(w).Encode(singleData)
				if err != nil {
					logErrors("encode", "JSON encoding failed", singleData.String())
				}
			} else {
				w.WriteHeader(http.StatusConflict)
				fmt.Fprintf(w, "NOOP: Data exists.")
			}
		}
	}
}

func deleteData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]
	found := false

	for i, singleData := range serviceData {
		if singleData.ID == dataID {
			found = true
			serviceData = append(serviceData[:i], serviceData[i+1:]...)
			fmt.Fprintf(w, "Data with ID %v has been deleted.", dataID)
		}
	}

	if !found {
		fmt.Fprintf(w, "Data with ID %v not found.", dataID)
	}
}

func initService() {
	if len(os.Args) < 2 {
		fmt.Println("FATAL: Service name not provided.")
		os.Exit(1)
	}

	serviceInfo = info{os.Args[1], serviceId.String()}

	hostName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	standardFields = log.Fields{
		"hostname": hostName,
		"service":  serviceInfo.NAME,
		"id":       serviceInfo.ID,
	}

	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(standardFields).WithFields(log.Fields{"args": os.Args, "mode": "init"}).Info("Service started successfully.")

	validate = validator.New()
}

func main() {

	initService()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", healthCheck)
	router.HandleFunc("/info", getServiceInfo).Methods("GET")
	router.HandleFunc("/data", getAllData).Methods("GET")
	router.HandleFunc("/data", createData).Methods("PUT")
	router.HandleFunc("/data/{id}", getData).Methods("GET")
	router.HandleFunc("/data/{id}", updateData).Methods("PATCH")
	router.HandleFunc("/data/{id}", deleteData).Methods("DELETE")

	log.WithFields(standardFields).WithFields(log.Fields{"mode": "run"}).Info("Listening on port 8080")

	fmt.Println(http.ListenAndServe(":8080", router))
}
