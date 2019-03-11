package birthday

import (
	"os"
	"os/signal"
	"time"

	"gobirthday/models"
	"gobirthday/providers"

	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

//------------------------------------------------------------------------------
// Structure
//------------------------------------------------------------------------------

// GoBirthday is a birthday reminder that allows you to not forget your loved ones.
type GoBirthday struct {
	contacts        []*models.Contact
	providers       []providers.Provider
	cronExp         string
	cron            *cron.Cron
	handleLeapYears bool
	runOnStartup    bool
}

//------------------------------------------------------------------------------
// Factory
//------------------------------------------------------------------------------

// NewGoBirthday returns new GoBirthday with the given CRON expression.
func NewGoBirthday(cronExp string, handleLeapYears, runOnStartup bool) *GoBirthday {
	return &GoBirthday{
		cron:            cron.New(),
		cronExp:         cronExp,
		handleLeapYears: handleLeapYears,
		runOnStartup:    runOnStartup,
	}
}

//------------------------------------------------------------------------------
// Functions
//------------------------------------------------------------------------------

// Notify notifies all the birthdays that need to be wished.
func (gb *GoBirthday) Notify() {
	// Process all the contacts
	for _, contact := range gb.contacts {
		// Check the birthdate
		if contact.IsBirthdayToday() {
			logrus.WithFields(logrus.Fields{
				"age":       contact.GetAge(),
				"firstname": contact.Firstname,
				"lastname":  contact.Lastname,
			}).Infoln("Birthday to wish !")

			// Send all the notifications
			for _, provider := range gb.providers {
				logrus.WithFields(logrus.Fields{
					"provider_type":   provider.Type(),
					"provider_vendor": provider.Vendor(),
				}).Infoln("Sending the notification")
				err := provider.SendNotification(contact)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"provider_type":   provider.Type(),
						"provider_vendor": provider.Vendor(),
					}).Errorln("Error while sending the notification :", err)
					continue
				}

				logrus.WithFields(logrus.Fields{
					"provider_type":   provider.Type(),
					"provider_vendor": provider.Vendor(),
				}).Infoln("Successfully sent the notification")
			}
		}

		// Check leap years
		if gb.handleLeapYears && contact.IsBornOnLeapYear() && time.Now().Day() == 1 && time.Now().Month() == time.March {
			logrus.WithFields(logrus.Fields{
				"age":       contact.GetAge(),
				"firstname": contact.Firstname,
				"lastname":  contact.Lastname,
			}).Infoln("Birthday to wish on a leap year !")
		}
	}
}

// NbContacts return the number of contacts.
func (gb *GoBirthday) NbContacts() int {
	return len(gb.contacts)
}

// NbProviders return the number of providers.
func (gb *GoBirthday) NbProviders() int {
	return len(gb.providers)
}

// Start starts the program and wait for OS signals.
func (gb *GoBirthday) Start() {
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)

	// Run on startup
	if gb.runOnStartup {
		gb.Notify()
	}

	// Add the function to the CRON
	logrus.WithFields(logrus.Fields{
		"cron_exp": gb.cronExp,
	}).Infoln("Adding function to the CRON")
	gb.cron.AddFunc(gb.cronExp, gb.Notify)

	// Start the CRON
	logrus.Infoln("Starting the CRON")
	gb.cron.Start()

	// Handle KILL or CTRL+C
	signal.Notify(signalChan, os.Kill, os.Interrupt)
	go func() {
		for range signalChan {
			logrus.Infoln("Received an interrupt, stopping services...")

			gb.cron.Stop()

			logrus.Infoln("Services stopped")

			cleanupDone <- true
		}
	}()

	logrus.Infoln("Waiting for birthdays to wish")

	<-cleanupDone
}
