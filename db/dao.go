package db

import (
	"errors"
	"reflect"
	"time"
)

func GetUserInfo(MemberID int) (interface{}, error) {
	parms := []interface{}{
		MemberID,
	}
	result, err := ExecSP("up_Get_LiveUser @MemberID=?", parms)
	if err != nil {
		return nil, err
	}
	data := result.([]map[string]interface{})
	if len(data) > 0 {
		return data[0], nil
	} else {
		return nil, errors.New("Data Not Found")
	}
}

func SetPoint(MemberID int, Point int, CID int, CName string) (interface{}, error) {
	parms := []interface{}{
		MemberID,
		Point,
		CID,
		CName,
	}

	result, err := ExecSP("up_Set_LiveCosume @MemberID=?,@Point=?,@CID=?,@CName=?", parms)
	if err != nil {
		return nil, err
	}
	data := result.([]map[string]interface{})
	if len(data) > 0 {
		return data[0], nil //Result, LivePoint, LiveEndTime, IsLiveVIP
	} else {
		return nil, errors.New("Data Not Found")
	}
}

func ReflectField(src interface{}, dst interface{}) {
	resdata := src.(map[string]interface{})

	reflectValue := reflect.ValueOf(dst).Elem()
	reflectType := reflectValue.Type()

	for i := 0; i < reflectValue.NumField(); i++ {
		field := reflectValue.Field(i)
		fieldType := reflectType.Field(i)
		//log.Println(fieldType.Type.String())
		switch fieldType.Type.String() {
		case "int64":
			if v, ok := resdata[fieldType.Name].(int64); ok {
				fieldValue := reflect.ValueOf(v).Convert(field.Type())
				field.Set(fieldValue)
			}
		case "time.Time":
			if v, ok := resdata[fieldType.Name].(time.Time); ok {
				fieldValue := reflect.ValueOf(v).Convert(field.Type())
				field.Set(fieldValue)
				//field.FieldByName(fieldType.Name).Set(reflect.ValueOf(v))
			}
		case "string":
			if v, ok := resdata[fieldType.Name].(string); ok {
				fieldValue := reflect.ValueOf(v).Convert(field.Type())
				field.Set(fieldValue)
			}
		case "float64":
			if v, ok := resdata[fieldType.Name].(float64); ok {
				fieldValue := reflect.ValueOf(v).Convert(field.Type())
				field.Set(fieldValue)
			}
		case "bool":
			if v, ok := resdata[fieldType.Name].(bool); ok {
				fieldValue := reflect.ValueOf(v).Convert(field.Type())
				field.Set(fieldValue)
			}
		}

	}
}
