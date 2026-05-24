package m365

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

type CalendarSyncService struct {
	client  *CalendarClient
	jobRepo *repository.JobRepository
	posRepo *repository.PositionRepository
	empRepo *repository.JobEmployeeRepository
	db      *repository.Database
	baseURL string
}

func NewCalendarSyncService(
	client *CalendarClient,
	jobRepo *repository.JobRepository,
	posRepo *repository.PositionRepository,
	empRepo *repository.JobEmployeeRepository,
	db *repository.Database,
	baseURL string,
) *CalendarSyncService {
	return &CalendarSyncService{
		client:  client,
		jobRepo: jobRepo,
		posRepo: posRepo,
		empRepo: empRepo,
		db:      db,
		baseURL: baseURL,
	}
}

// SyncAllEmployeeEvents aktualisiert die Kalendereinträge aller zugewiesenen Mitarbeiter.
// Wird als goroutine aufgerufen — Job-Daten müssen vollständig sein.
func (s *CalendarSyncService) SyncAllEmployeeEvents(jobID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		log.Printf("[CalendarSync] job %d not found: %v", jobID, err)
		return
	}
	if job.StartDate == nil {
		return
	}
	employees, err := s.empRepo.ListForJob(jobID)
	if err != nil {
		log.Printf("[CalendarSync] list employees for job %d: %v", jobID, err)
		return
	}
	for _, je := range employees {
		s.syncOne(job, je)
	}
}

// DeleteAllEmployeeEvents löscht alle Kalendereinträge für einen Job (beim Job-Löschen).
func (s *CalendarSyncService) DeleteAllEmployeeEvents(jobID uint) {
	employees, err := s.empRepo.ListForJob(jobID)
	if err != nil {
		return
	}
	for _, je := range employees {
		s.deleteOne(je)
	}
}

// SyncEmployeeEvent erstellt oder aktualisiert den Kalendereintrag für einen einzelnen Mitarbeiter.
func (s *CalendarSyncService) SyncEmployeeEvent(jobID, employeeID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil || job.StartDate == nil {
		return
	}
	je, err := s.empRepo.GetOne(jobID, employeeID)
	if err != nil {
		log.Printf("[CalendarSync] get job_employee %d/%d: %v", jobID, employeeID, err)
		return
	}
	s.syncOne(job, *je)
}

// DeleteEmployeeEvent löscht den Kalendereintrag eines Mitarbeiters für einen Job.
func (s *CalendarSyncService) DeleteEmployeeEvent(jobID, employeeID uint) {
	je, err := s.empRepo.GetOne(jobID, employeeID)
	if err != nil {
		return
	}
	s.deleteOne(*je)
}

func (s *CalendarSyncService) syncOne(job *models.Job, je models.JobEmployee) {
	if je.Employee.Email == nil || *je.Employee.Email == "" {
		return
	}
	email := *je.Employee.Email
	event, err := s.buildEvent(job)
	if err != nil {
		log.Printf("[CalendarSync] build event for job %d: %v", job.JobID, err)
		return
	}
	if je.M365EventID != nil && *je.M365EventID != "" {
		if err := s.client.UpdateUserEvent(email, *je.M365EventID, *event); err != nil {
			log.Printf("[CalendarSync] update event for employee %d job %d: %v", je.EmployeeID, job.JobID, err)
		}
		return
	}
	eventID, err := s.client.CreateUserEvent(email, *event)
	if err != nil {
		log.Printf("[CalendarSync] create event for employee %d job %d: %v", je.EmployeeID, job.JobID, err)
		return
	}
	if err := s.empRepo.SaveM365EventID(je.JobID, je.EmployeeID, eventID); err != nil {
		log.Printf("[CalendarSync] save event id for employee %d job %d: %v", je.EmployeeID, job.JobID, err)
	}
}

func (s *CalendarSyncService) deleteOne(je models.JobEmployee) {
	if je.M365EventID == nil || *je.M365EventID == "" || je.Employee.Email == nil {
		return
	}
	if err := s.client.DeleteUserEvent(*je.Employee.Email, *je.M365EventID); err != nil {
		log.Printf("[CalendarSync] delete event for employee %d: %v", je.EmployeeID, err)
		return
	}
	s.empRepo.ClearM365EventID(je.JobID, je.EmployeeID)
}

// SyncJobEvent — rückwärtskompatibel, delegiert an SyncAllEmployeeEvents.
func (s *CalendarSyncService) SyncJobEvent(jobID uint) {
	s.SyncAllEmployeeEvents(jobID)
}

// DeleteJobEvent — rückwärtskompatibel, delegiert an DeleteAllEmployeeEvents.
func (s *CalendarSyncService) DeleteJobEvent(jobID uint) {
	s.DeleteAllEmployeeEvents(jobID)
}

func (s *CalendarSyncService) buildEvent(job *models.Job) (*CalendarEvent, error) {
	positions, err := s.posRepo.GetByJobID(job.JobID)
	if err != nil {
		return nil, fmt.Errorf("load positions: %w", err)
	}
	jEmployees, err := s.empRepo.ListForJob(job.JobID)
	if err != nil {
		return nil, fmt.Errorf("load employees: %w", err)
	}

	desc := ""
	if job.Description != nil {
		desc = *job.Description
	}
	customerName := job.Customer.GetDisplayName()
	subject := fmt.Sprintf("%s - %s (%s)", desc, customerName, job.JobCode)

	body := s.buildBody(positions, job.JobID)

	start := job.StartDate.Format("2006-01-02") + "T00:00:00"
	var end string
	if job.EndDate != nil {
		end = job.EndDate.Add(24 * time.Hour).Format("2006-01-02") + "T00:00:00"
	} else {
		end = job.StartDate.Add(24 * time.Hour).Format("2006-01-02") + "T00:00:00"
	}

	var attendees []Attendee
	for _, je := range jEmployees {
		if je.Employee.Email != nil && *je.Employee.Email != "" {
			attendees = append(attendees, Attendee{
				EmailAddress: EmailAddr{Address: *je.Employee.Email},
				Type:         "required",
			})
		}
	}

	return &CalendarEvent{
		Subject:   subject,
		Body:      EventBody{ContentType: "HTML", Content: body},
		Start:     EventDateTime{DateTime: start, TimeZone: "Europe/Berlin"},
		End:       EventDateTime{DateTime: end, TimeZone: "Europe/Berlin"},
		Attendees: attendees,
	}, nil
}

func (s *CalendarSyncService) buildBody(positions []models.JobPosition, jobID uint) string {
	type section struct {
		label string
		types []string
	}
	sections := []section{
		{"Dienstleistungen", []string{"service"}},
		{"Produkte", []string{"product"}},
		{"Mietprodukte", []string{"rental"}},
	}

	var sb strings.Builder
	for i, sec := range sections {
		if i > 0 {
			sb.WriteString("<br>")
		}
		sb.WriteString(fmt.Sprintf("<b>%s</b><br>", sec.label))
		found := false
		for _, p := range positions {
			if sliceContains(sec.types, p.PositionType) {
				name := p.Description
				if name == "" && p.ProductID != nil {
					name = fmt.Sprintf("Produkt #%d", *p.ProductID)
				}
				qty := p.Quantity
				if qty == 0 {
					qty = 1
				}
				sb.WriteString(fmt.Sprintf("%s – %.0fx<br>", name, qty))
				found = true
			}
		}
		if !found {
			sb.WriteString("–<br>")
		}
	}

	if s.baseURL != "" {
		sb.WriteString(fmt.Sprintf(
			"<br><a href=\"%s/jobs/%d\">Job in RentalCore öffnen</a>",
			s.baseURL, jobID,
		))
	}
	return sb.String()
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
