package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"

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

type allData map[string]data

func (a allData) String() string {
	returnData := ""

	for _, x := range a {
		returnData += "{" + x.String() + "}"
	}

	return "[" + returnData + "]"
}

func (a allData) searchData(dataID string) (data, bool) {
	d, ok := a[dataID]
	return d, ok
}

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

const (
	incorrectInputError = "Please check submission: {\"ID\":\"<ID_VALUE>\",\"Message\":\"<MESSAGE_VALUE>\"}"
)

func logErrors(event string, message string, errorData ...string) {
	_, fileName, line, _ := runtime.Caller(1)
	log.WithFields(log.Fields{"event": event, "line": line, "file": fileName, "data": errorData}).Error(message)
}

type InfoController struct {
	serviceInfo info
}

func (ic InfoController) healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func (ic InfoController) getServiceInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, ic.serviceInfo.String())
}

type Logic struct {
	serviceData allData
	m           sync.Mutex
}

func (l *Logic) AddData(newData data) error {
	l.m.Lock()
	defer l.m.Unlock()
	if _, ok := l.serviceData.searchData(newData.ID); !ok {
		return ErrExists
	}
	l.serviceData[newData.ID] = newData
	return nil
}

func (l *Logic) Get(dataID string) (data, bool) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.serviceData.searchData(dataID)
}

func (l *Logic) GetAll() allData {
	l.m.Lock()
	defer l.m.Unlock()
	// returning a copy so that we don't modify without the lock
	out := allData{}
	for k, v := range l.serviceData {
		out[k] = v
	}
	return out
}

func (l *Logic) Update(updatedData data) (data, error) {
	l.m.Lock()
	defer l.m.Unlock()

	singleData, ok := l.serviceData[updatedData.ID]
	if !ok {
		return singleData, ErrNotFound
	}
	if singleData == updatedData {
		return singleData, ErrNoChange
	}
	l.serviceData[updatedData.ID] = updatedData
	return singleData, nil
}

func (l *Logic) Delete(dataID string) error {
	l.m.Lock()
	defer l.m.Unlock()
	if _, ok := l.serviceData[dataID]; !ok {
		return ErrNotFound
	}
	delete(l.serviceData, dataID)
	return nil
}

var (
	ErrExists   = errors.New("already exists")
	ErrNoChange = errors.New("same value")
	ErrNotFound = errors.New("not found")
)

func logError(err error, newData data) {
	var invaldErr *validator.InvalidValidationError
	if errors.As(err, &invaldErr) {
		logErrors("validate-errors", "", err.Error())
	}

	var multiErr validator.ValidationErrors
	if errors.As(err, &multiErr) {
		for _, err := range err.(validator.ValidationErrors) {
			logErrors("validate-errors", fmt.Sprintf("[%s,%s]", err.Field(), err.Tag()))
		}
	}

	logErrors("invalid-data-struct", "", newData.String())
}

type Controller struct {
	l        *Logic
	validate *validator.Validate
}

func (c *Controller) createData(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var newData data
	err := json.NewDecoder(r.Body).Decode(&newData)
	if err != nil {
		logErrors("unmarshal", "", newData.String())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(incorrectInputError))
		return
	}

	err = c.validate.Struct(newData)
	if err != nil {
		logError(err, newData)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, incorrectInputError)
		return
	}

	err = c.l.AddData(newData)

	if err != nil {
		if errors.Is(err, ErrExists) {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "ERROR: Data exists.")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "ERROR: %v", err)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (c *Controller) getData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]

	// return empty data structure if not found? should that be a 404 instead?
	foundData, ok := c.l.Get(dataID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(foundData)
	if err != nil {
		logErrors("encode", "JSON Encoding in getData", foundData.String())
	}
}

func (c *Controller) getAllData(w http.ResponseWriter, r *http.Request) {
	allData := c.l.GetAll()
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(allData)
	if err != nil {
		logErrors("unmarshal", "", allData.String())
	}
}

func (c *Controller) updateData(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	dataID := mux.Vars(r)["id"]
	var updatedData data

	err := json.NewDecoder(r.Body).Decode(&updatedData)
	if err != nil {
		logErrors("unmarshal", "", updatedData.String())
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, incorrectInputError)
		return
	}

	// need to have a single point of truth on id, should be the path var
	// or eliminate the path var
	updatedData.ID = dataID

	err = c.validate.Struct(updatedData)
	if err != nil {
		logError(err, updatedData)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, incorrectInputError)
		return
	}

	singleData, err := c.l.Update(updatedData)
	if err != nil {
		if errors.Is(err, ErrNoChange) {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "NOOP: Data exists.")
			return
		}
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "NOOP: %s not found.", updatedData.ID)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(singleData)
	if err != nil {
		logErrors("encode", "JSON encoding failed", singleData.String())
	}
}

func (c *Controller) deleteData(w http.ResponseWriter, r *http.Request) {
	dataID := mux.Vars(r)["id"]

	if err := c.l.Delete(dataID); err != nil {
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Data with ID %v not found.", dataID)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Unknown error occurred.")
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Data with ID %v has been deleted.", dataID)
}

func initService() (info, log.Fields) {
	if len(os.Args) < 2 {
		fmt.Println("FATAL: Service name not provided.")
		os.Exit(1)
	}

	serviceId := uuid.New()
	serviceInfo := info{os.Args[1], serviceId.String()}

	hostName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	standardFields := log.Fields{
		"hostname": hostName,
		"service":  serviceInfo.NAME,
		"id":       serviceInfo.ID,
	}

	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(standardFields).WithFields(log.Fields{"args": os.Args, "mode": "init"}).Info("Service started successfully.")

	return serviceInfo, standardFields
}

func main() {

	serviceInfo, standardFields := initService()

	l := Logic{
		serviceData: allData{},
	}

	validate := validator.New()
	c := Controller{
		l:        &l,
		validate: validate,
	}

	ic := InfoController{
		serviceInfo: serviceInfo,
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", ic.healthCheck).Methods("GET")
	router.HandleFunc("/info", ic.getServiceInfo).Methods("GET")
	router.HandleFunc("/data", c.getAllData).Methods("GET")
	router.HandleFunc("/data", c.createData).Methods("PUT") // should be POST
	router.HandleFunc("/data/{id}", c.getData).Methods("GET")
	router.HandleFunc("/data/{id}", c.updateData).Methods("PATCH")
	router.HandleFunc("/data/{id}", c.deleteData).Methods("DELETE")

	log.WithFields(standardFields).WithFields(log.Fields{"mode": "run"}).Info("Listening on port 8080")

	fmt.Println(http.ListenAndServe(":8080", router))
}
