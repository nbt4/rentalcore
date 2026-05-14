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

// SyncJobEvent erstellt oder aktualisiert den Kalendertermin für einen Job.
// Wird als goroutine aufgerufen — nie direkt Result prüfen.
func (s *CalendarSyncService) SyncJobEvent(jobID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		log.Printf("[CalendarSync] job %d not found: %v", jobID, err)
		return
	}
	if job.StartDate == nil {
		log.Printf("[CalendarSync] job %d has no start date, skipping", jobID)
		return
	}

	event, err := s.buildEvent(job)
	if err != nil {
		log.Printf("[CalendarSync] build event for job %d: %v", jobID, err)
		return
	}

	if job.M365EventID != nil && *job.M365EventID != "" {
		if err := s.client.UpdateEvent(*job.M365EventID, *event); err != nil {
			log.Printf("[CalendarSync] update event for job %d: %v", jobID, err)
		}
		return
	}

	eventID, err := s.client.CreateEvent(*event)
	if err != nil {
		log.Printf("[CalendarSync] create event for job %d: %v", jobID, err)
		return
	}
	if err := s.db.Model(&models.Job{}).Where("jobid = ?", jobID).
		Update("m365_event_id", eventID).Error; err != nil {
		log.Printf("[CalendarSync] save event id for job %d: %v", jobID, err)
	}
}

// DeleteJobEvent löscht den Kalendertermin und leert m365_event_id.
func (s *CalendarSyncService) DeleteJobEvent(jobID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil || job.M365EventID == nil || *job.M365EventID == "" {
		return
	}
	if err := s.client.DeleteEvent(*job.M365EventID); err != nil {
		log.Printf("[CalendarSync] delete event for job %d: %v", jobID, err)
		return
	}
	s.db.Model(&models.Job{}).Where("jobid = ?", jobID).Update("m365_event_id", nil)
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
