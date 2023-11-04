package hr_service

import (
	"log"

	"github.com/zapscloud/golib-dbutils/db_common"
	"github.com/zapscloud/golib-dbutils/db_utils"
	"github.com/zapscloud/golib-hr-repository/hr_common"
	"github.com/zapscloud/golib-hr-repository/hr_repository"
	"github.com/zapscloud/golib-platform-repository/platform_common"
	"github.com/zapscloud/golib-platform-repository/platform_repository"
	"github.com/zapscloud/golib-platform-service/platform_service"
	"github.com/zapscloud/golib-utils/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReportsService - Reports Service structure
type ReportsService interface {
	GetAttendanceSummary(filter string, aggr string, sort string, skip int64, limit int64) (utils.Map, error)

	BeginTransaction()
	CommitTransaction()
	RollbackTransaction()

	EndService()
}

type reportsBaseService struct {
	db_utils.DatabaseService
	dbRegion            db_utils.DatabaseService
	daoReports          hr_repository.ReportsDao
	daoPlatformBusiness platform_repository.BusinessDao
	daoPlatformAppUser  platform_repository.AppUserDao

	child      ReportsService
	businessID string
	staffID    string // Changed "staffId" to "staffID" for consistency
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
}

func NewReportsService(props utils.Map) (ReportsService, error) {
	funcode := hr_common.GetServiceModuleCode() + "M" + "01"

	log.Printf("ReportsService::Start ")

	// Verify whether the business id data passed
	businessID, err := utils.GetMemberDataStr(props, hr_common.FLD_BUSINESS_ID)
	if err != nil {
		return nil, err
	}

	p := reportsBaseService{} // Initialize p as a pointer to the struct

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

	// Verify whether the User id data passed, this is optional parameter
	staffID, _ := utils.GetMemberDataStr(props, hr_common.FLD_STAFF_ID)

	// Assign the BusinessID
	p.businessID = businessID
	p.staffID = staffID

	// Instantiate other services
	p.daoReports = hr_repository.NewReportsDao(p.dbRegion.GetClient(), p.businessID, p.staffID)
	p.daoPlatformBusiness = platform_repository.NewBusinessDao(p.GetClient())
	p.daoPlatformAppUser = platform_repository.NewAppUserDao(p.GetClient())

	_, err = p.daoPlatformBusiness.Get(businessID)
	if err != nil {
		err := &utils.AppError{
			ErrorCode:   funcode + "01",
			ErrorMsg:    "Invalid business_id",
			ErrorDetail: "Given app_business_id does not exist"}
		return p.errorReturn(err)
	}

	p.child = &p // Assign the pointer to itself

	return &p, err
}

func (p *reportsBaseService) EndService() {
	log.Printf("EndReportsMongoService ")
	p.CloseDatabaseService()
	p.dbRegion.CloseDatabaseService()
}

// GetAttendanceSummary retrieves reports data
func (p *reportsBaseService) GetAttendanceSummary(filter string, aggr string, sort string, skip int64, limit int64) (utils.Map, error) {
	log.Println("ReportsService::GetReportsData - Begin")

	daoReports := p.daoReports
	response, err := daoReports.GetAttendanceSummary(filter, aggr, sort, skip, limit)
	if err != nil {
		return nil, err
	}

	// Lookup Appuser Info
	p.lookupAppuser(response)

	log.Println("ReportsService::GetAttendanceSummary - End")
	return response, nil
}

// errorReturn handles error and closes the database connection
func (p *reportsBaseService) errorReturn(err error) (ReportsService, error) {
	// Close the Database Connection
	p.EndService()
	return nil, err
}

func (p *reportsBaseService) lookupAppuser(response utils.Map) {

	// Enumerate All staffs and lookup platform_app_user table
	dataStaff, err := utils.GetMemberData(response, db_common.LIST_RESULT)

	if err == nil {
		recs := dataStaff.([]utils.Map)
		for _, rec := range recs {
			data, err := utils.GetMemberData(rec, hr_common.FLD_GROUP_DOCS)
			if err == nil {
				docs := []interface{}(data.(primitive.A))
				for _, doc := range docs {
					p.mergeUserInfo(doc.(utils.Map))
				}
			}
		}
	}
}

func (p *reportsBaseService) mergeUserInfo(staffInfo utils.Map) {

	staffId, _ := utils.GetMemberDataStr(staffInfo, hr_common.FLD_STAFF_ID)

	staffData, err := p.daoPlatformAppUser.Get(staffId)
	if err == nil {
		// Delete unwanted fields
		delete(staffData, db_common.FLD_CREATED_AT)
		delete(staffData, db_common.FLD_UPDATED_AT)
		delete(staffData, platform_common.FLD_APP_USER_ID)

		// Make it as Array for backward compatible, since all MongoDB Lookups data returned as array
		staffInfo[hr_common.FLD_STAFF_INFO] = []utils.Map{staffData}
	}
}
