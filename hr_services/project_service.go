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

// ProjectService - Projects Service structure
type ProjectService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(projectId string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(projectId string, indata utils.Map) (utils.Map, error)
	Delete(projectId string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// projectBaseService - Projects Service structure
type projectBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoProject          hr_repository.ProjectDao
	daoPlatformBusiness platform_repository.BusinessDao
	child               ProjectService
	businessID          string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewProjectService(props utils.Map) (ProjectService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("ProjectService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := projectBaseService{}

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
	p.daoProject = hr_repository.NewProjectDao(p.dbRegion.GetClient(), p.businessID)
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

func (p *projectBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *projectBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("ProjectService::FindAll - Begin")

	daoProject := p.daoProject
	response, err := daoProject.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("ProjectService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *projectBaseService) Get(projectId string) (utils.Map, error) {
	log.Printf("ProjectService::FindByCode::  Begin %v", projectId)

	data, err := p.daoProject.Get(projectId)
	log.Println("ProjectService::FindByCode:: End ", err)
	return data, err
}

func (p *projectBaseService) Find(filter string) (utils.Map, error) {
	log.Println("ProjectService::FindByCode::  Begin ", filter)

	data, err := p.daoProject.Find(filter)
	log.Println("ProjectService::FindByCode:: End ", data, err)
	return data, err
}

func (p *projectBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var projectId string

	dataval, dataok := indata[hr_common.FLD_PROJECT_ID]
	if dataok {
		projectId = strings.ToLower(dataval.(string))
	} else {
		projectId = utils.GenerateUniqueId("projt")
		log.Println("Unique Project ID", projectId)
	}
	indata[hr_common.FLD_PROJECT_ID] = projectId
	indata[hr_common.FLD_BUSINESS_ID] = p.businessID
	log.Println("Provided Project ID:", projectId)

	_, err := p.daoProject.Get(projectId)
	if err == nil {
		err := &utils.AppError{ErrorCode: "S30102", ErrorMsg: "Existing Project ID !", ErrorDetail: "Given Project ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoProject.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *projectBaseService) Update(projectId string, indata utils.Map) (utils.Map, error) {

	log.Println("ProjectService::Update - Begin")

	data, err := p.daoProject.Get(projectId)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_PROJECT_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoProject.Update(projectId, indata)
	log.Println("ProjectService::Update - End ")
	return data, err
}

// Delete - Delete Service
func (p *projectBaseService) Delete(projectId string, delete_permanent bool) error {

	log.Println("ProjectService::Delete - Begin", projectId)

	daoProject := p.daoProject
	if delete_permanent {
		result, err := daoProject.Delete(projectId)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := p.Update(projectId, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("ProjectService::Delete - End")
	return nil
}

func (p *projectBaseService) errorReturn(err error) (ProjectService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
