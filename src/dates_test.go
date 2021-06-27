package main

import (
	"testing"
	"time"
)

func Test_GetTicketWorkdays(t *testing.T) {
	monday, friday := GetTicketWorkdays(1)

    //Checks for monday
	if weekday := monday.Weekday(); weekday != time.Weekday(1) {
		t.Errorf("Day of week incorrect, got: %v, want: %v.", weekday, time.Weekday(1))
	}
    if hour := monday.Hour(); hour != 9 {
        t.Errorf("Start hour is incorrect, got: %v, want: %v.", hour, 9)
    }

    
    //Checks for friday
	if weekday := friday.Weekday(); weekday != time.Weekday(5) {
		t.Errorf("Day of week incorrect, got: %v, want: %v.", weekday, time.Weekday(5))
	}

    if hour := friday.Hour(); hour != 17 {
        t.Errorf("End hour is incorrect, got: %v, want: %v.", hour, 17)
    }

}

