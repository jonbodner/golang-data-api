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
)

type data struct {
	ID      string `json:"ID"`
	Message string `json:"Message"`
}

func (d data) String() string {
	return fmt.Sprintf("ID:%s,Message:%s", d.ID, d.Message)
}

var incorrectInputError string = "Please check submission: {\"ID\":\"<ID_VALUE>\",\"Message\":\"<MESSAGE_VALUE>\"}"

var serviceName string

var serviceId uuid.UUID = uuid.New()

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

var standardFields log.Fields

func logErrors(event string, message string, errorData ...string) {
	_, fileName, line, _ := runtime.Caller(1)
	log.WithFields(log.Fields{"event": event, "line": line, "file": fileName, "data": errorData}).Error(message)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func serviceInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Service Name: %s, Service Instance ID: %s", serviceName, serviceId.String())
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
	}

	err = json.Unmarshal(input, &newData)
	if err != nil {
		logErrors("unmarshal", incorrectInputError, newData.String())
		fmt.Fprint(w, incorrectInputError)
	}

	foundData := searchData(newData.ID)

	if foundData == emptyData {
		serviceData = append(serviceData, newData)

		err = json.NewEncoder(w).Encode(newData)
		if err != nil {
			logErrors("encode", "JSON Encoding in createData", newData.String())
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
		logErrors("unmarshal", incorrectInputError, serviceData.String())
	}
}

func updateData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]
	var updatedData data

	input, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logErrors("read", incorrectInputError)
		fmt.Fprint(w, incorrectInputError)
	}

	err = json.Unmarshal(input, &updatedData)
	if err != nil {
		logErrors("unmarshal", incorrectInputError, updatedData.String())
		fmt.Fprint(w, incorrectInputError)
	}

	for i, singleData := range serviceData {
		if singleData.ID == dataID {
			if singleData != updatedData {
				singleData.Message = updatedData.Message
				serviceData = append(serviceData[:i], singleData)
				//w.WriteHeader(http.StatusOK)
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

	serviceName = os.Args[1]

	hostName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	standardFields = log.Fields{
		"hostname": hostName,
		"service":  serviceName,
		"id":       serviceId.String(),
	}

	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(standardFields).WithFields(log.Fields{"args": os.Args, "mode": "init"}).Info("Service started successfully.")
}

func main() {

	initService()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", healthCheck)
	router.HandleFunc("/info", serviceInfo)
	router.HandleFunc("/data", getAllData).Methods("GET")
	router.HandleFunc("/data", createData).Methods("PUT")
	router.HandleFunc("/data/{id}", getData).Methods("GET")
	router.HandleFunc("/data/{id}", updateData).Methods("PATCH")
	router.HandleFunc("/data/{id}", deleteData).Methods("DELETE")

	log.WithFields(standardFields).WithFields(log.Fields{"mode": "run"}).Info("Listening on port 8080")

	http.ListenAndServe(":8080", router)
}
