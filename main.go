package main

import (
	"encoding/json"
	"fmt"
	Log "github.com/sirupsen/logrus"
	"io/ioutil"
	"jimmyray.io/data-api/api"
	"jimmyray.io/data-api/utils"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	utils.Logger.WithFields(utils.StandardFields).WithFields(Log.Fields{"mode": "run"}).Debug("Listening on port 8080")
	_, _ = fmt.Fprintln(w, "OK")
}

func getServiceInfo(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, api.ServiceInfo.String())
}

func createData(w http.ResponseWriter, r *http.Request) {
	var output api.Data

	input, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorData := utils.ErrorLog{Skip: 1, Event: api.HttpReqReadErr, Message: err.Error(), ErrorData: string(input)}
		utils.LogErrors(errorData)
		_, _ = fmt.Fprint(w, api.IncorrectInputErr)
		return
	}

	output, err = api.CreateData(input)
	if err != nil {
		if err.Error() == api.DataConflictErr {
			_, _ = fmt.Fprint(w, api.DataConflictErr)
		} else {
			_, _ = fmt.Fprint(w, api.IncorrectInputErr)
		}
	} else {
		err = json.NewEncoder(w).Encode(output)
		if err != nil {
			errorData := utils.ErrorLog{Skip: 1, Event: api.JsonEncodeErr, Message: err.Error(), ErrorData: output.String()}
			utils.LogErrors(errorData)
		}
	}
}

func getData(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	data := api.GetData(id)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		errorData := utils.ErrorLog{Skip: 1, Event: api.JsonEncodeErr, Message: err.Error(), ErrorData: data.String()}
		utils.LogErrors(errorData)
	}
}

func getAllData(w http.ResponseWriter, r *http.Request) {
	data := api.GetAllData()
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		errorData := utils.ErrorLog{Skip: 1, Event: api.JsonEncodeErr, Message: err.Error(), ErrorData: data.String()}
		utils.LogErrors(errorData)
	}
}

func patchData(w http.ResponseWriter, r *http.Request) {
	input, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorData := utils.ErrorLog{Skip: 1, Event: api.HttpReqReadErr, Message: err.Error(), ErrorData: string(input)}
		utils.LogErrors(errorData)
		_, _ = fmt.Fprint(w, api.IncorrectInputErr)
		return
	}

	var data api.Data
	data, err = api.UpdateData(input)

	if err != nil {
		switch err.Error() {
		case api.NoDataFoundErr:
			_, _ = fmt.Fprintf(w, "Data could not be patched: %s", api.NoDataFoundErr)
		case api.DataConflictErr:
			_, _ = fmt.Fprintf(w, "Data could not be patched: %s", api.DataConflictErr)
		default:
			_, _ = fmt.Fprintf(w, "Data could not be patched: %s", api.IncorrectInputErr)
		}
	} else {
		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			errorData := utils.ErrorLog{Skip: 1, Event: api.HttpReqReadErr, Message: err.Error(), ErrorData: data.String()}
			utils.LogErrors(errorData)
		}
	}
}

func deleteData(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := api.DeleteData(id)
	if err != nil {
		_, _ = fmt.Fprintf(w, "Data was not deleted: %s", err.Error())
	} else {
		_, _ = fmt.Fprintf(w, "Data with ID %v has been deleted.", id)
	}
}

func initService() {
	if len(os.Args) < 2 {
		fmt.Println("FATAL: Service name not provided.")
		os.Exit(1)
	}

	api.ServiceInfo.NAME = os.Args[1]
	api.ServiceInfo.ID = api.GetServiceId()

	hostName, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	fields := Log.Fields{
		"hostname": hostName,
		"service":  api.ServiceInfo.NAME,
		"id":       api.ServiceInfo.ID,
	}

	var level Log.Level
	if len(os.Args) >= 3 {
		switch os.Args[2] {
		case "debug":
			level = Log.DebugLevel
		case "error":
			level = Log.ErrorLevel
		case "fatal":
			level = Log.FatalLevel
		case "warn":
			level = Log.WarnLevel
		default:
			level = Log.InfoLevel
		}
	} else {
		level = Log.InfoLevel
	}

	utils.InitLogs(fields, level)

	utils.Logger.WithFields(utils.StandardFields).WithFields(Log.Fields{"args": os.Args, "mode": "init", "logLevel": level}).Info("Service started successfully.")

	api.InitValidator()
}

func main() {
	initService()
	utils.Logger.WithFields(utils.StandardFields).WithFields(Log.Fields{"mode": "run"}).Info("Listening on port 8080")

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/healthz", healthCheck)
	router.HandleFunc("/info", getServiceInfo).Methods("GET")
	router.HandleFunc("/data", getAllData).Methods("GET")
	router.HandleFunc("/data", createData).Methods("PUT")
	router.HandleFunc("/data/{id}", getData).Methods("GET")
	router.HandleFunc("/data/{id}", patchData).Methods("PATCH")
	router.HandleFunc("/data/{id}", deleteData).Methods("DELETE")

	fmt.Println(http.ListenAndServe(":8080", router))
}
