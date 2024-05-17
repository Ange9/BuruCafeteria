package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Vendor    string
	EntryTime time.Time
	ExitTime  time.Time
	TotalWork time.Duration
}

func main() {
	// Open the CSV file
	file, err := os.Open("Report_1_240516145656.csv")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Parse the CSV file with pipe as delimiter
	reader := csv.NewReader(file)
	reader.Comma = '|' // Set the delimiter to pipe
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	// Parse records
	var totalWorkMinutes int
	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {

			vendor := row[0]
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			total := row[3]

			totalWorkDayMinutes, _ := parseTotalTimeToMinutes(total)

			// Subtract 60 minutes for lunch break
			totalWorkDayMinutes -= 60

			// Calculate payment for the day
			payment := calculatePayment(totalWorkDayMinutes)

			// Output result
			fmt.Printf("Vendor: %s, Date: %s, Payment: $%.2f\n", vendor, entryTime.Format("2006-01-02"), payment)

			// Aggregate total work time in minutes
			totalWorkMinutes += totalWorkDayMinutes
		}
	}

	// Output total work time in hours and minutes
	totalHours := totalWorkMinutes / 60
	totalMinutes := totalWorkMinutes % 60
	fmt.Printf("Total work time: %dh %dm\n", totalHours, totalMinutes)
}

func parseTotalTimeToMinutes(total string) (int, error) {
	parts := strings.Split(total, " ")
	hours, err := strconv.Atoi(strings.TrimSuffix(parts[0], "h"))
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(strings.TrimSuffix(parts[1], "m"))
	if err != nil {
		return 0, err
	}
	return hours*60 + minutes, nil
}

func calculatePayment(totalWorkMinutes int) float64 {
	hourlyPay := 1600.0 / 60 // Hourly pay converted to pay per minute
	extraTimePay := 1.5 * hourlyPay

	// Calculate payment
	if totalWorkMinutes > 8*60 {
		// Calculate extra time payment
		extraMinutes := totalWorkMinutes - 8*60
		return 8*60*hourlyPay + float64(extraMinutes)*extraTimePay
	}
	return float64(totalWorkMinutes) * hourlyPay
}
