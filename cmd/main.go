package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kr/pretty"
	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-service/hr_service"
	"github.com/zapscloud/golib-utils/utils"
)

func GetDBCreds() utils.Map {
	dbtype := db_common.DATABASE_TYPE_MONGODB
	dbuser := os.Getenv("MONGO_DB_USER")
	dbsecret := os.Getenv("MONGO_DB_SECRET")
	dbserver := os.Getenv("MONGO_DB_SERVER")
	dbname := os.Getenv("MONGO_DB_NAME")

	dbCreds := utils.Map{
		db_common.DB_TYPE:   dbtype,
		db_common.DB_SERVER: dbserver,
		db_common.DB_NAME:   dbname,
		db_common.DB_USER:   dbuser,
		db_common.DB_SECRET: dbsecret}

	return dbCreds
}

func MdbMain(businessid string) hr_service.AttendanceService {

	dbCreds := GetDBCreds()

	log.Println("DB Credentials: ", dbCreds)

	if dbCreds[db_common.DB_SERVER].(string) == "" {
		log.Println("Environment variable MONGO_DB_SERVER should be defined")
		return nil
	} else if dbCreds[db_common.DB_NAME].(string) == "" {
		log.Println("Environment variable MONGO_DB_NAME should be defined")
		return nil
	}

	dbCreds[hr_common.FLD_BUSINESS_ID] = businessid

	rolesrv, err := hr_service.NewAttendanceService(dbCreds)
	log.Println("User Mongo Service Error ", err)
	return rolesrv
}

func main() {

	//businessid := "business003"
	//rolesrv := MdbMain(businessid)
	// usersrv, bizsrv, rolesrv := ZapsMain(businessid)

	// EmptyBusiness(bizsrv)
	// DeleteBusiness(bizsrv)
	// CreateBusiness(bizsrv)
	// GetBusiness(bizsrv)

	//if rolesrv != nil {
	//EmptyBusinessAttendance(rolesrv)
	// DeleteAttendance(rolesrv)
	// CreateAttendance(rolesrv)
	// UpdateAttendance(rolesrv)
	//ListAttendances(rolesrv)
	// GetAttendance(rolesrv)
	// FindAttendance(rolesrv)
	//}
	loc, _ := time.LoadLocation("Asia/Calcutta")
	date_time := "2023-09-12 09:00:00 AM"
	layout := "2006-01-02 03:04:05 PM"
	dateResult, err := time.ParseInLocation(layout, date_time, loc)
	log.Println("DateTime ", dateResult, dateResult.UTC(), err)

	fromTime := "21:00:00"
	frTime, _ := time.Parse(time.TimeOnly, fromTime)

	toTime := "18:00:00"
	tTime, _ := time.Parse(time.TimeOnly, toTime)
	if frTime.Unix() < tTime.Unix() {
		log.Println("From Time is smaller")
	} else {
		log.Println("To Time is smaller")
	}

	log.Println(frTime.Unix(), tTime.Unix())

}

func EmptyBusinessAttendance(srv hr_service.AttendanceService) {
	log.Println("Attendance Service ")
}

func CreateAttendance(srv hr_service.AttendanceService) {

	indata := utils.Map{
		"role_id":    "role003",
		"role_name":  "Demo Attendance 003",
		"role_scope": "admin",
	}

	res, err := srv.ClockIn(indata)
	log.Println("Create Attendance", err)
	pretty.Println(res)

}

func GetAttendance(srv hr_service.AttendanceService) {
	res, err := srv.Get("role001")
	log.Println("Get Attendance", err)
	pretty.Println(res)

}

func FindAttendance(srv hr_service.AttendanceService) {

	filter := fmt.Sprintf(`{"%s":"%s"}`, "role_scope", "admin")
	res, err := srv.Find(filter)
	log.Println("Get Attendance", err)
	pretty.Println(res)

}

func UpdateAttendance(srv hr_service.AttendanceService) {

	indata := utils.Map{
		"role_id":   "role001",
		"role_name": "Demo Attendance 001 Updated",
		"is_active": true,
	}

	res, err := srv.Update("role001", indata)
	log.Println("Update Attendance", err)
	pretty.Println(res)

}

func DeleteAttendance(srv hr_service.AttendanceService) {

	srv.BeginTransaction()
	err := srv.Delete("role001", false)
	log.Println("DeleteAttendance success ", err)
	log.Println("DeleteAttendance Value ")

	if err != nil {
		srv.RollbackTransaction()
	} else {
		srv.CommitTransaction()
	}
}

func ListAttendances(srv hr_service.AttendanceService) {

	filter := "" //fmt.Sprintf(`{"%s":"%s"}`, "role_scope", "admin")

	sort := `{ "role_scope":1, "role_id":1}`

	res, err := srv.List(filter, sort, 0, 0)
	log.Println("List User success ", err)
	log.Println("List User summary ", res)
	pretty.Print(res)
}
