package facades

import (
	"github.com/goravel/framework/contracts/schedule"
)

// Schedule returns the scheduler facade.
func Schedule() schedule.Schedule {
	return App().MakeSchedule()
}
