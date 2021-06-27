package main

import (
	"log"
	"time"
)

//getTicketWorkdays returns Monday and Friday of the week after weeksOffset weeks
func GetTicketWorkdays(weeksOffset int)(time.Time, time.Time) {

	now := time.Now()

	days := 7 * weeksOffset
	desiredWeek := now.AddDate(0,0,days)
	
	dayOfWeek := desiredWeek.Weekday()

	//Moday of desired week
	desiredWeekStart := desiredWeek.AddDate(0,0, -int(dayOfWeek)+1)
	//Friday of desired week
	desiredWeekEnd := desiredWeekStart.AddDate(0,0, 4)
	
	//Load default location
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatal("Error in parsing dates")
	}
	
	//Set default start time(9am)
	desiredWeekStart = time.Date(desiredWeekStart.Year(), desiredWeekStart.Month(), desiredWeekStart.Day(), 9,0,0,0, location)
	//Set default end time(5pm)
	desiredWeekEnd = time.Date(desiredWeekEnd.Year(), desiredWeekEnd.Month(), desiredWeekEnd.Day(), 17,0,0,0, location)
	
	return desiredWeekStart, desiredWeekEnd
	

}
