package hr_service

import (
	"log"
	"strings"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
)

// WorkLocationService - Accounts Service structure
type WorkLocationService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(workLocId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(workLocId string, indata utils.Map) (utils.Map, error)
	Delete(workLocId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// workLocationBaseService - Accounts Service structure
type workLocationBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoWorkLocation     hr_repository.WorkLocationDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               WorkLocationService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewWorkLocationService(props utils.Map) (WorkLocationService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("WorkLocationService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := workLocationBaseService{}

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

	// Assign the BusinessId
	p.businessID = businessId

	// Instantiate other services
	p.daoWorkLocation = hr_repository.NewWorkLocationDao(p.dbRegion.GetClient(), p.businessID)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())

	_, err = p.daoPlatformBusiness.Get(p.businessID)
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

func (p *workLocationBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *workLocationBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("AccountService::FindAll - Begin")

	daoWorkLocation := p.daoWorkLocation
	response, err := daoWorkLocation.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("AccountService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *workLocationBaseService) Get(workLocId string) (utils.Map, error) {
	log.Printf("AccountService::FindByCode::  Begin %v", workLocId)

	data, err := p.daoWorkLocation.Get(workLocId)
	log.Println("AccountService::FindByCode:: End ", err)
	return data, err
}

func (p *workLocationBaseService) Find(filter string) (utils.Map, error) {
	log.Println("AccountService::FindByCode::  Begin ", filter)

	data, err := p.daoWorkLocation.Find(filter)
	log.Println("AccountService::FindByCode:: End ", data, err)
	return data, err
}

func (p *workLocationBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var holidayId string

	dataval, dataok := indata[hr_common.FLD_WORKLOCATION_ID]
	if dataok {
		holidayId = strings.ToLower(dataval.(string))
	} else {
		holidayId = utils.GenerateUniqueId("wrkloc")
		log.Println("Unique Account ID", holidayId)
	}
	indata[hr_common.FLD_WORKLOCATION_ID] = holidayId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Account ID:", holidayId)

	_, err := p.daoWorkLocation.Get(holidayId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Account ID !", ErrorDetail: "Given Account ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoWorkLocation.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *workLocationBaseService) Update(workLocId string, indata utils.Map) (utils.Map, error) {

	log.Println("AccountService::Update - Begin")

	data, err := p.daoWorkLocation.Get(workLocId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_WORKLOCATION_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoWorkLocation.Update(workLocId, indata)
	log.Println("AccountService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *workLocationBaseService) Delete(workLocId string, delete_permanent bool) error {

	log.Println("AccountService::Delete - Begin", workLocId)

	daoWorkLocation := p.daoWorkLocation
	_, err := daoWorkLocation.Get(workLocId)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoWorkLocation.Delete(workLocId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoWorkLocation.Update(workLocId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("WorkLocationService::Delete - End")
	return nil
}

func (p *workLocationBaseService) errorReturn(err error) (WorkLocationService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
