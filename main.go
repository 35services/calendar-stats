// calendar-tracker, a program to compute statistics from Google calendars.
// Copyright (C) 2023 Marcin Owsiany <marcin@owsiany.pl>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/porridge/calendar-tracker/internal/config"
	"github.com/porridge/calendar-tracker/internal/core"
	"github.com/porridge/calendar-tracker/internal/io"
	"github.com/porridge/calendar-tracker/internal/ordererd"
	"google.golang.org/api/calendar/v3"
)

var notice string = `
Copyright (C) 2023 Marcin Owsiany <marcin@owsiany.pl>
Copyright (C) Google Inc.
Copyright (C) 2011 Google LLC.
This program comes with ABSOLUTELY NO WARRANTY.
This is free software, and you are welcome to redistribute it under the terms
of the terms of the GNU General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version.

Google Calendar is a trademark of Google LLC.
`

func main() {
	configFile := flag.String("config", "config.yaml", "Name of configuration file to read.")
	source := flag.String("source", "primary", "Name of Google Calendar to read.")
	weekCount := flag.Int("weeks", 0, "How many weeks before the current one to look at.")
	cacheFileName := flag.String("cache", "", "If not empty, name of json file to use as event cache. "+
		"If file does not exist, it will be created and fetched events will be stored there. "+
		"Otherwise, events will be loaded from this file rather than fetched from Google Calendar.")
	decimalOutput := flag.Bool("decimal-output", false, "If true, print totals as decimal fractions rather than XhYmZs Duration format.")

	origUsage := flag.Usage
	flag.Usage = func() {
		origUsage()
		fmt.Fprint(flag.CommandLine.Output(), notice)
	}

	flag.Parse()
	events, err := io.GetEvents(*source, *weekCount, *cacheFileName)
	if err != nil {
		log.Fatalf("Failed to retrieve events: %s", err)
	}

	categories, err := config.Read(*configFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(events.Items) == 0 {
		fmt.Println("No events found.")
	} else {
		dayTotals, categoryTotals, unrecognized := core.ComputeTotals(events, categories)
		days := ordererd.KeysOfMap(dayTotals, ordererd.CivilDates)
		var total time.Duration
		if len(days) > 0 {
			fmt.Println("Time spent per day:")
		}
		for _, day := range days {
			total += dayTotals[day]
			value := formatDayTotal(decimalOutput, dayTotals[day])
			fmt.Printf("%v: %s\n", day, value)
		}
		if len(categories) > 0 {
			fmt.Println("Time spent per category:")
		}
		for _, category := range categories {
			catName := category.Name
			val := categoryTotals[catName]
			fraction := (int64(val) * 100) / int64(total)
			if catName == core.Uncategorized {
				catName = "(uncategorized)"
			}
			fmt.Printf("%2d%% %s\n", fraction, catName)
		}
		if len(unrecognized) > 0 {
			fmt.Println("Unrecognized:")
		}
		for _, un := range unrecognized {
			fmt.Println(formatUnrecognizedEvent(un))
		}
	}
}

func formatUnrecognizedEvent(event *calendar.Event) string {
	if event.Start == nil || event.End == nil {
		return "?"
	}
	start, err1 := time.Parse(time.RFC3339, event.Start.DateTime)
	end, err2 := time.Parse(time.RFC3339, event.End.DateTime)
	if err1 != nil || err2 != nil {
		return "?"
	}
	return fmt.Sprintf("%s %10s  %s", event.Start.DateTime, end.Sub(start).String(), event.Summary)
}

func formatDayTotal(decimalOutput *bool, d time.Duration) string {
	if *decimalOutput {
		return fmt.Sprintf("%f", float64(d)/float64(time.Hour))
	} else {
		return d.String()
	}
}
