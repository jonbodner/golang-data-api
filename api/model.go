package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"jimmyray.io/data-api/utils"

	"github.com/google/uuid"

	"github.com/go-playground/validator/v10"
)

const (
	HttpReqReadErr			string = "HTTP_REQ_READ_ERR"
	IncorrectInputErr		string = "Please check submission: {\"ID\":\"<ID_VALUE>\",\"Message\":\"<MESSAGE_VALUE>\"}"
	JsonEncodeErr			string = "JSON_ENCODE_ERR"
	JsonParseErr     		string = "JSON_PARSE_ERR"
	JsonUnmarshalErr  		string = "JSON_UNMARSHAL_ERR"
	JsonNotWellformedErr    string = "JSON_NOT_WELLFORMED_ERR"
	//JsonValidateErr			string = "JSON_VALIDATE_ERR"
	DataConflictErr     	string = "NOOP_DATA_CONFLICT_ERR"
	NoDataFoundErr    		string = "NO_DATA_FOUND_ERR"
	InvalidValidationErr	string = "INVALID_VALIDATION_ERR"
	ValidationErr			string = "VALIDATION_ERR"
	InvalidDataStructErr	string = "INVALID_DATA_STRUCT_ERR"
)

var Validate *validator.Validate

func InitValidator() {
	Validate = validator.New()
}

var serviceId = uuid.New()

func GetServiceId() string {
	return serviceId.String()
}

type Data struct {
	ID      string `json:"ID" validate:"required"`
	Message string `json:"Message" validate:"required"`
}

func (d Data) Json() string {
	out, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return string(out)
}

func (d Data) String() string {
	out := fmt.Sprintf("{ID: %s, Message: %s}", d.ID, d.Message)
	return out
}

var emptyData = Data{}

type AllData []Data

func (a AllData) String() string {
	returnData := ""

	for _, x := range serviceData {
		returnData += "{" + x.Json() + "}"
	}

	return "[" + returnData + "]"
}

var serviceData AllData

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

var ServiceInfo = info{}

func searchData(dataID string) (foundData Data) {
	for _, x := range serviceData {
		if x.ID == dataID {
			foundData = x
		}
	}

	return
}

func CreateData(input []byte) (Data, error) {
	var newData Data

	if !json.Valid(input) {
		return newData, errors.New(JsonParseErr)
	}

	err := json.Unmarshal(input, &newData)
	if err != nil {
		fmt.Println(err)
		return newData, errors.New(JsonUnmarshalErr)
	}

	err = Validate.Struct(newData)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			errorData := utils.ErrorLog{Skip: 1, Event: InvalidValidationErr, Message: err.Error()}
			utils.LogErrors(errorData)
		}

		for _, err := range err.(validator.ValidationErrors) {
			errorData := utils.ErrorLog{Skip: 1, Event: ValidationErr, Message: err.Error(), ErrorData: fmt.Sprintf("[%s,%s]", err.Field(), err.Tag())}
			utils.LogErrors(errorData)
		}

		errorData := utils.ErrorLog{Skip: 1, Event: InvalidDataStructErr, ErrorData: newData.String()}
		utils.LogErrors(errorData)

		return newData, errors.New(InvalidDataStructErr)
	}

	foundData := searchData(newData.ID)

	if foundData == emptyData {
		serviceData = append(serviceData, newData)

		return newData, nil
	} else {
		return newData, errors.New(DataConflictErr)
	}
}

func GetData(id string) Data {
	return searchData(id)
}

func GetAllData() AllData {
	return serviceData
}

func UpdateData(input []byte) (Data, error) {
	if !json.Valid(input) {
		errorData := utils.ErrorLog{Skip: 1, Event: JsonNotWellformedErr, ErrorData: string(input)}
		utils.LogErrors(errorData)
		return Data{}, errors.New(JsonNotWellformedErr)
	}

	var updatedData Data

	err := json.Unmarshal(input, &updatedData)
	if err != nil {
		//logErrors("unmarshal", "", updatedData.String())
		return Data{}, errors.New(JsonUnmarshalErr)
	}

	err = Validate.Struct(updatedData)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			errorData := utils.ErrorLog{Skip: 1, Event: InvalidValidationErr, Message: err.Error()}
			utils.LogErrors(errorData)
		}

		for _, err := range err.(validator.ValidationErrors) {
			errorData := utils.ErrorLog{Skip: 1, Event: ValidationErr, Message: err.Error(), ErrorData: fmt.Sprintf("[%s,%s]", err.Field(), err.Tag())}
			utils.LogErrors(errorData)
		}

		errorData := utils.ErrorLog{Skip: 1, Event: InvalidDataStructErr, ErrorData: updatedData.String()}
		utils.LogErrors(errorData)

		return updatedData, errors.New(InvalidDataStructErr)
	}

	foundData := searchData(updatedData.ID)
	if foundData == emptyData {
		return emptyData, errors.New(NoDataFoundErr)
	}

	for i, singleData := range serviceData {
		if singleData.ID == updatedData.ID {
			if singleData != updatedData {
				singleData.Message = updatedData.Message
				serviceData = append(serviceData[:i], singleData)
			} else {
				return Data{}, errors.New(DataConflictErr)
			}
		}
	}

	return updatedData, nil
}

func DeleteData(id string) error {
	found := false

	for i, singleData := range serviceData {
		if singleData.ID == id {
			found = true
			serviceData = append(serviceData[:i], serviceData[i+1:]...)
		}
	}

	if found {
		return nil
	} else {
		return errors.New(NoDataFoundErr)
	}
}

//func LoadData(overwrite bool, data AllData) error {
//
//	if overwrite {
//		serviceData = data
//		return nil
//	} else {
//		for i, singleData := range data {
//
//		}
//	}
//
//}
