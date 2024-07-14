package main

//
// This program downloads the rendered fx rates HTML pages from the Central Bank of
// Seychelles (CBS) site and prints out the rates for SCR in USD, EUR, and GBP.
//
// Created by Emile O. E. Antat <eoea754@gmail.com>
//

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/playwright-community/playwright-go"
)

// hasCurrDateRates: takes the file path and returns true if the file
// modification date is the same as the current date; false otherwise.
func hasCurrDateRates(ratesFile string) bool {
	fileInfo, err := os.Stat(ratesFile)

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	f1, f2, f3 := fileInfo.ModTime().Date()
	t1, t2, t3 := time.Now().Date()

	return f1 == t1 && f2 == t2 && f3 == t3
}

// fetchCBSRates: gets the Central Bank of Seychelles rates for USD, EUR, and
// GBP and returns the content as an HTML string.
func fetchCBSRates() string {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Firefox.Launch()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{IgnoreHttpsErrors: playwright.Bool(true)})
	if err != nil {
		log.Fatalf("Could not create new context: %v", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		log.Fatalf("Could not create page: %v", err)
	}
	if _, err := page.Goto("https://www.cbs.sc/marketinfo/DailyRates.html"); err != nil {
		log.Fatalf("Could not goto: %v", err)
	}
	content, err := page.Content()
	if err != nil {
		log.Fatalf("Could not get content: %v", err)
	}
	return content
}

// extractRates: takes a currency and a rendered HTML with the rates information
// and returns the HTML section for the specified rate.
//
// In the regex statement, the number is the number of lines (or section) about
// the information that I need such as the selling, buying and mid-rates for the
// respective currency. Currency in this specific ratesHTML is GBP, EUR, or USD.
func extractRates(curr string, ratesHTML string) string {
	s := fmt.Sprintf(".*%s.*(\n.*?){4}", curr)
	rates, err := regexp.Compile(s)
	if err != nil {
		log.Fatalf("Failed to compile regex: %v", err)
	}
	section := rates.FindAllString(ratesHTML, -1)[0]
	return section
}

// prettyPrint: Takes the section of the rates after extractRates() and prints
// out the information on the rates that I need in a convenient layout.
func prettyPrint(rates string) {
	pattern := `<th style="height: 30px;font-size: 12px">(\w+)</th>\s+<td style="font-size: 12px;text-align: left" class="ng-binding">(\d+\.\d+)</td>\s+<td style="font-size: 12px;text-align: left" class="ng-binding">(\d+\.\d+)</td>\s+<td style="font-size: 12px;text-align: left" class="ng-binding">(\d+\.\d+)</td>`

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(rates, -1)

	if len(matches) > 0 {
		fmt.Println("Currency:", matches[0][1])
		fmt.Println("Buying:  ", matches[0][2])
		fmt.Println("Selling: ", matches[0][3])
		fmt.Println("Mid-rate:", matches[0][4])
		fmt.Println()
	} else {
		// TODO(eoea):
		// This will usually return on GBP if there is no Selling or Mid-Rate
		// price. For the time being I decided not to implement this because I
		// don't have a lot of GBP payment.
		fmt.Println("No rates found.")
	}
}

func main() {
	ratesFile := "/tmp/cbsrates.html"
	ratesHTML := ""

	day := time.Now().Weekday()

	// CBS does not seem to update their rates on Saturdays and Sundays, so the
	// request times out if we run this on those days; this is the fix to ignore
	// downloads on Saturdays and Sundays. This has not been tested on Public
	// Holidays.
	if day != time.Saturday && day != time.Sunday {
		if !hasCurrDateRates(ratesFile) {
			ratesHTML = fetchCBSRates()
			err := os.WriteFile(ratesFile, []byte(ratesHTML), 0644)
			if err != nil {
				log.Fatalf("Failed to write to temporary file: %v", err)
			}
		}
	}

	if len(ratesHTML) == 0 {
		content, err := os.ReadFile(ratesFile)
		if err != nil {
			log.Fatalf("Could not read an old rates file: %v from %s", err, ratesFile)
		}
		ratesHTML = string(content)
	}

	prettyPrint(extractRates("USD", ratesHTML))
	prettyPrint(extractRates("EUR", ratesHTML))
	prettyPrint(extractRates("GBP", ratesHTML))
}
