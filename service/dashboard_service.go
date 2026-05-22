package service

import (
	"service-travego/model"
	"service-travego/repository"
	"time"
)

type DashboardService struct {
	repo *repository.DashboardRepository
}

func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{
		repo: repo,
	}
}

func (s *DashboardService) GetPartnerSummary(orgID string) (*model.DashboardPartnerSummaryResponse, error) {
	return s.repo.GetPartnerSummary(orgID)
}

func (s *DashboardService) GetDashboard(orgID string) (*model.DashboardResponse, error) {
	return s.repo.GetDashboard(orgID)
}

func (s *DashboardService) GetTopDestinations(orgID string) ([]model.DashboardTopDestination, error) {
	return s.repo.GetTopDestinations(orgID)
}

func (s *DashboardService) GetTopPickupCity(orgID string) ([]model.DashboardTopPickupCity, error) {
	return s.repo.GetTopPickupCity(orgID)
}

func (s *DashboardService) GetTopFleets(orgID string) ([]model.DashboardTopFleet, error) {
	return s.repo.GetTopFleets(orgID)
}

func (s *DashboardService) GetTopTourPackages(orgID string) ([]model.DashboardTopTourPackage, error) {
	return s.repo.GetTopTourPackages(orgID)
}

func (s *DashboardService) GetTopDrivers(orgID string) ([]model.DashboardTopDriver, error) {
	return s.repo.GetTopDrivers(orgID)
}

func (s *DashboardService) GetTopCustomers(orgID string) ([]model.DashboardTopCustomer, error) {
	return s.repo.GetTopCustomers(orgID)
}

type DashboardFinanceResponse struct {
	GroupBy string                  `json:"group_by"`
	Labels  []string                `json:"labels"`
	Series  []DashboardFinanceSerie `json:"series"`
	Summary DashboardFinanceSummary `json:"summary"`
}

type DashboardFinanceSerie struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

type DashboardFinanceSummary struct {
	TotalRevenue  float64 `json:"total_revenue"`
	TotalExpenses float64 `json:"total_expenses"`
	Net           float64 `json:"net"`
}

func (s *DashboardService) GetFinance(orgID string, startDate time.Time, endDate time.Time) (*DashboardFinanceResponse, error) {
	diffDays := int(endDate.Sub(startDate).Hours() / 24)

	groupBy := "month"
	if diffDays <= 14 {
		groupBy = "day"
	} else if diffDays <= 30 {
		groupBy = "2day"
	} else if diffDays <= 180 {
		groupBy = "week"
	}

	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.Local)
	endDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.Local)
	end := endDay.AddDate(0, 0, 1).Add(-time.Nanosecond)

	rows, err := s.repo.GetFinance(orgID, groupBy, start, end)
	if err != nil {
		return nil, err
	}

	var startPeriod time.Time
	var endPeriod time.Time
	labelFormat := "2006-01"

	switch groupBy {
	case "day":
		startPeriod = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
		endPeriod = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, time.Local)
		labelFormat = "2006-01-02"
	case "2day":
		startPeriod = truncateTo2DayBucket(start)
		endPeriod = truncateTo2DayBucket(endDay)
		labelFormat = "2006-01-02"
	case "week":
		startPeriod = truncateToWeek(start)
		endPeriod = truncateToWeek(endDay)
		labelFormat = "2006-01-02"
	default:
		startPeriod = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)
		endPeriod = time.Date(endDay.Year(), endDay.Month(), 1, 0, 0, 0, 0, time.Local)
		labelFormat = "2006-01"
	}

	byLabel := make(map[string][2]float64, len(rows))
	for _, r := range rows {
		period := r.Period.In(time.Local)
		label := period.Format(labelFormat)
		byLabel[label] = [2]float64{r.Revenue, r.Expenses}
	}

	labels := make([]string, 0)
	revenueData := make([]float64, 0)
	expensesData := make([]float64, 0)

	var totalRevenue float64
	var totalExpenses float64

	for t := startPeriod; !t.After(endPeriod); t = addPeriod(t, groupBy) {
		label := t.Format(labelFormat)
		labels = append(labels, label)

		v := byLabel[label]
		revenueData = append(revenueData, v[0])
		expensesData = append(expensesData, v[1])

		totalRevenue += v[0]
		totalExpenses += v[1]
	}

	return &DashboardFinanceResponse{
		GroupBy: groupBy,
		Labels:  labels,
		Series: []DashboardFinanceSerie{
			{Name: "Revenue", Data: revenueData},
			{Name: "Expenses", Data: expensesData},
		},
		Summary: DashboardFinanceSummary{
			TotalRevenue:  totalRevenue,
			TotalExpenses: totalExpenses,
			Net:           totalRevenue - totalExpenses,
		},
	}, nil
}

func truncateToWeek(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	weekday := int(d.Weekday())
	offset := (weekday + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func truncateTo2DayBucket(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	if d.YearDay()%2 == 1 {
		return d.AddDate(0, 0, -1)
	}
	return d
}

func addPeriod(t time.Time, groupBy string) time.Time {
	switch groupBy {
	case "day":
		return t.AddDate(0, 0, 1)
	case "2day":
		return t.AddDate(0, 0, 2)
	case "week":
		return t.AddDate(0, 0, 7)
	default:
		return t.AddDate(0, 1, 0)
	}
}
