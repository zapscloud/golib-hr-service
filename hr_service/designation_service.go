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

// DesignationService - Designations Service structure
type DesignationService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(designation_id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(designation_id string, indata utils.Map) (utils.Map, error)
	Delete(designation_id string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// DesignationBaseService - Designations Service structure
type designationBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoDesignation      hr_repository.DesignationDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               DesignationService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewDesignationService(props utils.Map) (DesignationService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"
	log.Printf("DesignationService::Start")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := designationBaseService{}

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

	// Assign the BusinessId
	p.businessID = businessId

	// Instantiate other services
	p.daoDesignation = hr_repository.NewDesignationDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *designationBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// ************************
// List - List All records
//
// ************************
func (p *designationBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("DesignationService::FindAll - Begin")

	daoDesignation := p.daoDesignation
	response, err := daoDesignation.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("DesignationService::FindAll - End ")
	return response, nil
}

// *************************
// Get - Get Details
//
// *************************
func (p *designationBaseService) Get(designation_id string) (utils.Map, error) {
	log.Printf("DesignationService::FindByCode::  Begin %v", designation_id)

	data, err := p.daoDesignation.Get(designation_id)
	log.Println("DesignationService::FindByCode:: End ", err)
	return data, err
}

// ************************
// Find - Find Service
//
// ************************
func (p *designationBaseService) Find(filter string) (utils.Map, error) {
	log.Println("DesignationService::FindByCode::  Begin ", filter)

	data, err := p.daoDesignation.Find(filter)
	log.Println("DesignationService::FindByCode:: End ", data, err)
	return data, err
}

// ************************
// Create - Create Service
//
// ************************
func (p *designationBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")
	var desigId string

	dataval, dataok := indata[hr_common.FLD_DESIGNATION_ID]
	if dataok {
		desigId = strings.ToLower(dataval.(string))
	} else {
		desigId = utils.GenerateUniqueId("desig")
		log.Println("Unique Designation ID", desigId)
	}
	indata[hr_common.FLD_DESIGNATION_ID] = desigId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Designation ID:", desigId)

	_, err := p.daoDesignation.Get(desigId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Designation ID !", ErrorDetail: "Given Designation ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoDesignation.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// ************************
// Update - Update Service
//
// ************************
func (p *designationBaseService) Update(designation_id string, indata utils.Map) (utils.Map, error) {

	log.Println("DesignationService::Update - Begin")

	data, err := p.daoDesignation.Get(designation_id)
	if err != nil {
		return data, err
	}

	// Delete the Key fields
	delete(indata, hr_common.FLD_DESIGNATION_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoDesignation.Update(designation_id, indata)
	log.Println("DesignationService::Update - End ")
	return data, err
}

// ************************
// Delete - Delete Service
//
// ************************
func (p *designationBaseService) Delete(designation_id string, delete_permanent bool) error {

	log.Println("DesignationService::Delete - Begin", designation_id, delete_permanent)

	daoDesignation := p.daoDesignation
	_, err := daoDesignation.Get(designation_id)
	if err != nil {
		return err
	}

	if delete_permanent {
		result, err := daoDesignation.Delete(designation_id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {

		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := daoDesignation.Update(designation_id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("DesignationService::Delete - End")
	return nil
}

func (p *designationBaseService) errorReturn(err error) (DesignationService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
