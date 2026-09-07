package m365

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"go-barcode-webapp/internal/logger"
)

type CalendarSyncService struct {
	client  *CalendarClient
	jobRepo *repository.JobRepository
	posRepo *repository.PositionRepository
	empRepo *repository.JobEmployeeRepository
	baseURL string
	mu      sync.Mutex
}

func NewCalendarSyncService(
	client *CalendarClient,
	jobRepo *repository.JobRepository,
	posRepo *repository.PositionRepository,
	empRepo *repository.JobEmployeeRepository,
	baseURL string,
) *CalendarSyncService {
	return &CalendarSyncService{
		client:  client,
		jobRepo: jobRepo,
		posRepo: posRepo,
		empRepo: empRepo,
		baseURL: baseURL,
	}
}

// SyncAllEmployeeEvents ist der rückwärtskompatible Einstiegspunkt für den
// gemeinsamen Job-Termin in der Kalender-Mailbox.
func (s *CalendarSyncService) SyncAllEmployeeEvents(jobID uint) {
	s.SyncJobEvent(jobID)
}

// DeleteAllEmployeeEvents ist der rückwärtskompatible Einstiegspunkt.
func (s *CalendarSyncService) DeleteAllEmployeeEvents(jobID uint) {
	s.DeleteJobEvent(jobID)
}

// SyncEmployeeEvent aktualisiert den einen Job-Termin und dessen Teilnehmerliste.
func (s *CalendarSyncService) SyncEmployeeEvent(jobID, employeeID uint) {
	s.SyncJobEvent(jobID)
}

// DeleteEmployeeEvent entfernt nur noch einen eventuell vorhandenen Alttermin.
// Die Teilnehmerliste des gemeinsamen Termins wird nach dem DB-Remove über
// SyncJobEvent aktualisiert.
func (s *CalendarSyncService) DeleteEmployeeEvent(jobID, employeeID uint) {
	je, err := s.empRepo.GetOne(jobID, employeeID)
	if err != nil {
		return
	}
	s.deleteLegacyEmployeeEvent(*je)
}

func (s *CalendarSyncService) deleteLegacyEmployeeEvent(je models.JobEmployee) {
	if je.M365EventID == nil || *je.M365EventID == "" || je.Employee.Email == nil {
		return
	}
	if err := s.client.DeleteUserEvent(*je.Employee.Email, *je.M365EventID); err != nil {
		if !isCalendarNotFound(err) {
			logger.LogInfo("[CalendarSync] delete legacy event for employee %d: %v", je.EmployeeID, err)
			return
		}
	}
	if err := s.empRepo.ClearM365EventID(je.JobID, je.EmployeeID); err != nil {
		logger.LogInfo("[CalendarSync] clear legacy event id for employee %d: %v", je.EmployeeID, err)
	}
}

func (s *CalendarSyncService) cleanupLegacyEmployeeEvents(employees []models.JobEmployee) {
	for _, employee := range employees {
		s.deleteLegacyEmployeeEvent(employee)
	}
}

// SyncJobEvent erstellt genau einen Termin in der Kalender-/Raum-Mailbox und
// hält dessen Teilnehmerliste mit den zugewiesenen Bearbeitern synchron.
func (s *CalendarSyncService) SyncJobEvent(jobID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		logger.LogInfo("[CalendarSync] job %d not found: %v", jobID, err)
		return
	}
	employees, err := s.empRepo.ListForJob(jobID)
	if err != nil {
		logger.LogInfo("[CalendarSync] list employees for job %d: %v", jobID, err)
		return
	}

	if job.StartDate == nil {
		s.deleteCentralEvent(job)
		s.cleanupLegacyEmployeeEvents(employees)
		return
	}

	event, err := s.buildEvent(job, employees)
	if err != nil {
		logger.LogInfo("[CalendarSync] build event for job %d: %v", jobID, err)
		return
	}

	if job.M365EventID != nil && *job.M365EventID != "" {
		err = s.client.UpdateEvent(*job.M365EventID, *event)
		if err == nil {
			s.cleanupLegacyEmployeeEvents(employees)
			if updated, getErr := s.client.GetEvent(*job.M365EventID); getErr != nil {
				logger.LogInfo("[CalendarSync] load shared event for attendee acceptance on job %d: %v", jobID, getErr)
			} else {
				s.scheduleAttendeeAcceptance(event.Attendees, updated.ICalUID)
			}
			return
		}
		if !isCalendarNotFound(err) {
			logger.LogInfo("[CalendarSync] update shared event for job %d: %v", jobID, err)
			return
		}
		if clearErr := s.jobRepo.ClearM365EventID(jobID); clearErr != nil {
			logger.LogInfo("[CalendarSync] clear missing event id for job %d: %v", jobID, clearErr)
			return
		}
	}

	event.TransactionID = jobTransactionID(job)
	created, err := s.client.CreateEvent(*event)
	if err != nil {
		logger.LogInfo("[CalendarSync] create shared event for job %d: %v", jobID, err)
		return
	}
	if err := s.jobRepo.SaveM365EventID(jobID, created.ID); err != nil {
		logger.LogInfo("[CalendarSync] save shared event id for job %d: %v", jobID, err)
		return
	}
	s.cleanupLegacyEmployeeEvents(employees)
	s.scheduleAttendeeAcceptance(event.Attendees, created.ICalUID)
}

func (s *CalendarSyncService) DeleteJobEvent(jobID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		return
	}
	employees, err := s.empRepo.ListForJob(jobID)
	if err == nil {
		s.cleanupLegacyEmployeeEvents(employees)
	}
	s.deleteCentralEvent(job)
}

func (s *CalendarSyncService) deleteCentralEvent(job *models.Job) {
	if job.M365EventID == nil || *job.M365EventID == "" {
		return
	}
	if err := s.client.DeleteEvent(*job.M365EventID); err != nil && !isCalendarNotFound(err) {
		logger.LogInfo("[CalendarSync] delete shared event for job %d: %v", job.JobID, err)
		return
	}
	if err := s.jobRepo.ClearM365EventID(job.JobID); err != nil {
		logger.LogInfo("[CalendarSync] clear shared event id for job %d: %v", job.JobID, err)
	}
}

func (s *CalendarSyncService) buildEvent(job *models.Job, employees []models.JobEmployee) (*CalendarEvent, error) {
	positions, err := s.posRepo.GetByJobID(job.JobID)
	if err != nil {
		return nil, fmt.Errorf("load positions: %w", err)
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
		end = job.EndDate.Add(24*time.Hour).Format("2006-01-02") + "T00:00:00"
	} else {
		end = job.StartDate.Add(24*time.Hour).Format("2006-01-02") + "T00:00:00"
	}

	location := buildLocation(job)

	return &CalendarEvent{
		Subject:               subject,
		Body:                  EventBody{ContentType: "HTML", Content: body},
		Start:                 EventDateTime{DateTime: start, TimeZone: "Europe/Berlin"},
		End:                   EventDateTime{DateTime: end, TimeZone: "Europe/Berlin"},
		Location:              location,
		Attendees:             buildAttendees(employees, s.client.mailbox),
		IsAllDay:              true,
		ShowAs:                "busy",
		ResponseRequested:     false,
		AllowNewTimeProposals: false,
	}, nil
}

func buildAttendees(employees []models.JobEmployee, calendarMailbox string) []Attendee {
	attendees := make([]Attendee, 0, len(employees))
	seen := make(map[string]struct{}, len(employees))
	excluded := strings.ToLower(strings.TrimSpace(calendarMailbox))
	for _, assignment := range employees {
		if assignment.Employee.Email == nil {
			continue
		}
		email := strings.TrimSpace(*assignment.Employee.Email)
		normalized := strings.ToLower(email)
		if email == "" || normalized == excluded {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		attendees = append(attendees, Attendee{
			EmailAddress: EmailAddr{Address: email},
			Type:         "required",
		})
	}
	return attendees
}

func jobTransactionID(job *models.Job) string {
	if job.CreatedAt != nil {
		return fmt.Sprintf("rentalcore-job-%d-%d", job.JobID, job.CreatedAt.UnixNano())
	}
	return fmt.Sprintf("rentalcore-job-%d", job.JobID)
}

func (s *CalendarSyncService) scheduleAttendeeAcceptance(attendees []Attendee, iCalUID string) {
	if iCalUID == "" {
		return
	}
	for _, attendee := range attendees {
		email := attendee.EmailAddress.Address
		go s.acceptAttendee(email, iCalUID)
	}
}

func (s *CalendarSyncService) acceptAttendee(email, iCalUID string) {
	delays := []time.Duration{0, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error
	for _, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		eventID, err := s.client.FindUserEventByICalUID(email, iCalUID)
		if err != nil {
			lastErr = err
			continue
		}
		if err := s.client.AcceptUserEvent(email, eventID); err != nil {
			lastErr = err
			continue
		}
		return
	}
	logger.LogInfo("[CalendarSync] auto-accept shared event for %s: %v", email, lastErr)
}

func buildLocation(job *models.Job) *EventLocation {
	if job.VenueID != nil && job.Venue != nil {
		v := job.Venue
		parts := []string{v.Name}
		street := ""
		if v.Street != nil && *v.Street != "" {
			street = *v.Street
		}
		if v.HouseNumber != nil && *v.HouseNumber != "" {
			street = strings.TrimSpace(street + " " + *v.HouseNumber)
		}
		if street != "" {
			parts = append(parts, street)
		}
		cityPart := ""
		if v.ZIP != nil && *v.ZIP != "" {
			cityPart = *v.ZIP
		}
		if v.City != nil && *v.City != "" {
			cityPart = strings.TrimSpace(cityPart + " " + *v.City)
		}
		if cityPart != "" {
			parts = append(parts, cityPart)
		}
		return &EventLocation{DisplayName: strings.Join(parts, ", ")}
	}

	c := job.Customer
	street := ""
	if c.Street != nil && *c.Street != "" {
		street = *c.Street
	}
	if c.HouseNumber != nil && *c.HouseNumber != "" {
		street = strings.TrimSpace(street + " " + *c.HouseNumber)
	}
	cityPart := ""
	if c.ZIP != nil && *c.ZIP != "" {
		cityPart = *c.ZIP
	}
	if c.City != nil && *c.City != "" {
		cityPart = strings.TrimSpace(cityPart + " " + *c.City)
	}
	var addrParts []string
	if street != "" {
		addrParts = append(addrParts, street)
	}
	if cityPart != "" {
		addrParts = append(addrParts, cityPart)
	}
	if len(addrParts) > 0 {
		return &EventLocation{DisplayName: strings.Join(addrParts, ", ")}
	}
	return nil
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
