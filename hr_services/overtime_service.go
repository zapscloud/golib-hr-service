package hr_services

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

// OvertimeService - Accounts Service structure
type OvertimeService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(overtimeId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(overtimeId string, indata utils.Map) (utils.Map, error)
	Delete(overtimeId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// OvertimeeBaseService - Accounts Service structure
type OvertimeBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoHrsFactor        hr_repository.OvertimeDao
	daoPlatformBusiness platform_repository.BusinessDao

	child      OvertimeService
	businessId string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewOvertimeService(props utils.Map) (OvertimeService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("OvertimeService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := OvertimeBaseService{}

	// Open Database Service
	err = p.OpenDatabaseService(props)
	if err != nil {
		return nil, err
	}

	// Open RegionDB Service
	p.dbRegion, err = platform_services.OpenRegionDatabaseService(props)
	if err != nil {
		p.CloseDatabaseService()
		return nil, err
	}

	// Assign the BusinessId & StaffId
	p.businessId = businessId

	// Instantiate other services
	p.daoHrsFactor = hr_repository.NewOvertimeDao(p.dbRegion.GetClient(), p.businessId)
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

func (p *OvertimeBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *OvertimeBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("OvertimeService::FindAll - Begin")

	daoHrsFactor := p.daoHrsFactor
	response, err := daoHrsFactor.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("OvertimeService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *OvertimeBaseService) Get(overtimeId string) (utils.Map, error) {
	log.Printf("OvertimeService::FindByCode::  Begin %v", overtimeId)

	data, err := p.daoHrsFactor.Get(overtimeId)
	log.Println("OvertimeService::FindByCode:: End ", err)
	return data, err
}

func (p *OvertimeBaseService) Find(filter string) (utils.Map, error) {
	log.Println("OvertimeService::FindByCode::  Begin ", filter)

	data, err := p.daoHrsFactor.Find(filter)
	log.Println("OvertimeService::FindByCode:: End ", data, err)
	return data, err
}

func (p *OvertimeBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var overtimeId string

	dataval, dataok := indata[hr_common.FLD_OVERTIME_ID]
	if dataok {
		overtimeId = strings.ToLower(dataval.(string))
	} else {
		overtimeId = utils.GenerateUniqueId("ot")
		log.Println("Unique OT ID", overtimeId)
	}
	indata[hr_common.FLD_OVERTIME_ID] = overtimeId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	log.Println("Provided OT ID:", overtimeId)

	_, err := p.daoHrsFactor.Get(overtimeId)
	if err == nil {
		err := &utils.AppError{
			ErrorCode:   "S30102",
			ErrorMsg:    "Existing Hours Factor ID !",
			ErrorDetail: "Given Hours Factor ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoHrsFactor.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *OvertimeBaseService) Update(overtimeId string, indata utils.Map) (utils.Map, error) {

	log.Println("OvertimeService::Update - Begin")

	data, err := p.daoHrsFactor.Get(overtimeId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_OVERTIME_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoHrsFactor.Update(overtimeId, indata)
	log.Println("OvertimeService::Update - End ", err)
	return data, err
}

// Delete - Delete Service
func (p *OvertimeBaseService) Delete(overtimeId string, delete_permanent bool) error {

	log.Println("OvertimeService::Delete - Begin", overtimeId)

	daoHrsFactor := p.daoHrsFactor
	if delete_permanent {
		result, err := daoHrsFactor.Delete(overtimeId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := p.Update(overtimeId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("OvertimeService::Delete - End")
	return nil
}

func (p *OvertimeBaseService) errorReturn(err error) (OvertimeService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
