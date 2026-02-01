package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

func getEmployee(config *domain.Configuration, key string) (domain.Employee, bool) {
	if config == nil || config.PersonalDetails == nil {
		return defaultEmployee(), false
	}
	employee, ok := config.PersonalDetails[key]
	if !ok {
		return defaultEmployee(), false
	}
	return employee, true
}

func defaultEmployee() domain.Employee {
	return domain.Employee{
		EmploymentType: domain.EmploymentTypeNonFederal,
		BirthDate:      time.Date(ProjectionBaseYear, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(ProjectionBaseYear, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
