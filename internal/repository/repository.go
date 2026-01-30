package repository

import "database/sql"

// Repository содержит все репозитории
type Repository struct {
	Client      *ClientRepository
	Admin       *AdminRepository
	Exercise    *ExerciseRepository
	Plan        *PlanRepository
	Appointment *AppointmentRepository
	Schedule    *ScheduleRepository
	Program     *ProgramRepository
}

// New создаёт новый экземпляр Repository
func New(db *sql.DB) *Repository {
	return &Repository{
		Client:      NewClientRepository(db),
		Admin:       NewAdminRepository(db),
		Exercise:    NewExerciseRepository(db),
		Plan:        NewPlanRepository(db),
		Appointment: NewAppointmentRepository(db),
		Schedule:    NewScheduleRepository(db),
		Program:     NewProgramRepository(db),
	}
}
