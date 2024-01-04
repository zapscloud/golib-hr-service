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

// VisaTypeService - Accounts Service structure
type VisaTypeService interface {
	List(filter string, sort string, skip int64, limit int64) (utils.Map, error)
	Get(visatype_Id string) (utils.Map, error)
	Find(filter string) (utils.Map, error)
	Create(indata utils.Map) (utils.Map, error)
	Update(visatype_Id string, indata utils.Map) (utils.Map, error)
	Delete(visatype_Id string, delete_permanent bool) error

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

// visatypeBaseService - Accounts Service structure
type visatypeBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoVisaType         hr_repository.VisaTypeDao
	daoPlatformBusiness platform_repository.BusinessDao

	child      VisaTypeService
	businessId string
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewVisaTypeService(props utils.Map) (VisaTypeService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("VisaTypeService::Start ")

	// Verify whether the business id data passed
	businessId, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := visatypeBaseService{}

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
	p.daoVisaType = hr_repository.NewVisaTypeDao(p.dbRegion.GetClient(), p.businessId)
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

func (p *visatypeBaseService) EndService() {
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// List - List All records
func (p *visatypeBaseService) List(filter string, sort string, skip int64, limit int64) (utils.Map, error) {

	log.Println("VisaTypeService::FindAll - Begin")

	daoVisaType := p.daoVisaType
	response, err := daoVisaType.List(filter, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	log.Println("VisaTypeService::FindAll - End ")
	return response, nil
}

// FindByCode - Find By Code
func (p *visatypeBaseService) Get(visatype_Id string) (utils.Map, error) {
	log.Printf("VisaTypeService::FindByCode::  Begin %v", visatype_Id)

	data, err := p.daoVisaType.Get(visatype_Id)
	log.Println("VisaTypeService::FindByCode:: End ", err)
	return data, err
}

func (p *visatypeBaseService) Find(filter string) (utils.Map, error) {
	log.Println("VisaTypeService::FindByCode::  Begin ", filter)

	data, err := p.daoVisaType.Find(filter)
	log.Println("VisaTypeService::FindByCode:: End ", data, err)
	return data, err
}

func (p *visatypeBaseService) Create(indata utils.Map) (utils.Map, error) {

	log.Println("UserService::Create - Begin")

	var visatype_Id string

	dataval, dataok := indata[hr_common.FLD_VISA_TYPE_ID]
	if dataok {
		visatype_Id = strings.ToLower(dataval.(string))
	} else {
		visatype_Id = utils.GenerateUniqueId("vity")
		log.Println("Unique Account ID", visatype_Id)
	}
	indata[hr_common.FLD_VISA_TYPE_ID] = visatype_Id
	indata[hr_common.FLD_BUSINESS_ID] = p.businessId
	log.Println("Provided Account ID:", visatype_Id)

	_, err := p.daoVisaType.Get(visatype_Id)
	if err == nil {
		err := &utils.AppError{
			ErrorCode:   "S30102",
			ErrorMsg:    "Existing VisaType ID !",
			ErrorDetail: "Given VisaType ID already exist"}
		return indata, err
	}

	insertResult, err := p.daoVisaType.Create(indata)
	if err != nil {
		return indata, err
	}
	log.Println("UserService::Create - End ", insertResult)
	return indata, err
}

// Update - Update Service
func (p *visatypeBaseService) Update(visatype_Id string, indata utils.Map) (utils.Map, error) {

	log.Println("VisaTypeService::Update - Begin")

	data, err := p.daoVisaType.Get(visatype_Id)
	if err != nil {
		return data, err
	}

	// Delete key fields
	delete(indata, hr_common.FLD_VISA_TYPE_ID)
	delete(indata, hr_common.FLD_BUSINESS_ID)

	data, err = p.daoVisaType.Update(visatype_Id, indata)
	log.Println("VisaTypeService::Update - End ", err)
	return data, err
}

// Delete - Delete Service
func (p *visatypeBaseService) Delete(visatype_Id string, delete_permanent bool) error {

	log.Println("VisaTypeService::Delete - Begin", visatype_Id)

	daoVisaType := p.daoVisaType
	if delete_permanent {
		result, err := daoVisaType.Delete(visatype_Id)
		if err != nil {
			return err
		}
		log.Printf("Delete %v", result)
	} else {
		indata := utils.Map{db_common.FLD_IS_DELETED: true}
		data, err := p.Update(visatype_Id, indata)
		if err != nil {
			return err
		}
		log.Println("Update for Delete Flag", data)
	}

	log.Printf("VisaTypeService::Delete - End")
	return nil
}

func (p *visatypeBaseService) errorReturn(err error) (VisaTypeService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}
