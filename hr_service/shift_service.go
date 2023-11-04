package hr_service

import (
	"log"
	"strings"
	"time"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// ShiftService - Accounts Service structure
type ShiftService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(shiftId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(shiftId string, indata utils.Map) (utils.Map, error)
	Delete(shiftId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// shiftBaseService - Accounts Service structure
type shiftBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoShift            hr_repository.ShiftDao
	daoPlatformBusiness platform_repository.BusinessDao

	child      ShiftService
	businessId string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewShiftService(props utils.Map) (ShiftService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("ShiftService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := shiftBaseService{}

	// Open Database Service
	err = p.OpenDatabaseService(props)
	if err != nil {
		return nil, err
	}

	// Open RegionDB Service
	p.dbRegion, err = platform_service.OpenRegionDatabaseService(props)
	if err != nil {
		p.CloseDatabaseService()
		return nil, err
	}

	// Assign the BusinessId & StaffId
	p.businessId = businessId

	// Instantiate other services
	p.daoShift = hr_repository.NewShiftDao(p.dbRegion.GetClient(), p.businessId)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())

	_, err = p.daoPlatformBusiness.Get(p.businessId)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business id",
			ErrorDetail: "Given business id is not exist"}
		return p.errorReturn(err)
	}

	p.child = &p

	return &p, nil
}

func (p *shiftBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *shiftBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("ShiftService::FindAll - Begin")

	daoShift := p.daoShift
	response, err := daoShift.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("ShiftService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *shiftBaseService) Get(shiftId string) (utils.Map, error) {
	log.Printf("ShiftService::FindByCode::  Begin %v", shiftId)

	data, err := p.daoShift.Get(shiftId)
	log.Println("ShiftService::FindByCode:: End ", err)
	return data, err
}

func (p *shiftBaseService) Find(filter string) (utils.Map, error) {
	log.Println("ShiftService::FindByCode::  Begin ", filter)

	data, err := p.daoShift.Find(filter)
	log.Println("ShiftService::FindByCode:: End ", data, err)
	return data, err
}

func (p *shiftBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var shiftId string

	dataval, dataok := indata[hr_common.FLD_SHIFT_ID]
	if dataok {
		shiftId = strings.ToLower(dataval.(string))
	} else {
		shiftId = utils.GenerateUniqueId("shift")
		log.Println("Unique Account ID", shiftId)
	}
	indata[hr_common.FLD_SHIFT_ID] = shiftId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	log.Println("Provided Account ID:", shiftId)

	_, err := p.daoShift.Get(shiftId)
	if err == nil {
		err := &utils.AppError{
			ErrorCode:   "S30102",
			ErrorMsg:    "Existing Shift ID !",
			ErrorDetail: "Given Shift ID already exist"}
		return indata, err
	}
	// Validate TimeFormat
	if p.validateTimeFormat(indata) != nil {
		return indata, err
	}

	insertResult, err := p.daoShift.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *shiftBaseService) Update(shiftId string, indata utils.Map) (utils.Map, error) {

	log.Println("ShiftService::Update - Begin")

	data, err := p.daoShift.Get(shiftId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_SHIFT_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	// Validate the TimeFormat
	if p.validateTimeFormat(indata) != nil {
		log.Println("ShiftService::convertStrToTimeFormat - Error ", err)
		return indata, err
	}

	data, err = p.daoShift.Update(shiftId, indata)
	log.Println("ShiftService::Update - End ", err)
	return data, err
}

// Delete - Delete Service
func (p *shiftBaseService) Delete(shiftId string, delete_permanent bool) error {

	log.Println("ShiftService::Delete - Begin", shiftId)

	daoShift := p.daoShift
	if delete_permanent {
		result, err := daoShift.Delete(shiftId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := p.Update(shiftId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("ShiftService::Delete - End")
	return nil
}

func (p *shiftBaseService) errorReturn(err error) (ShiftService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *shiftBaseService) validateTimeFormat(indata utils.Map) error {
	// Convert Time string to Date Format
	shiftFromTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_SHIFT_FROM)
	if err == nil {
		_, err = time.Parse(time.TimeOnly, shiftFromTime)
		if err != nil {
			err = &utils.AppError{
				ErrorCode:   "S30102",
				ErrorMsg:    "Failed to Parse Time Value",
				ErrorDetail: "Invalid From-Shift-Time value"}
			return err
		}
	}

	// Convert Time string to Date Format
	shiftToTime, err := utils.GetMemberDataStr(indata, hr_common.FLD_SHIFT_TO)
	if err == nil {
		_, err = time.Parse(time.TimeOnly, shiftToTime)
		if err != nil {
			err = &utils.AppError{
				ErrorCode:   "S30102",
				ErrorMsg:    "Failed to Parse Time Value",
				ErrorDetail: "Invalid To-Shift-Time value"}
			return err
		}
	}

	return nil
}
