package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
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

var overallTotalPayment float64 // Global variable to track total payment across all files

func main() {
	// Use filepath.Glob to match files with the naming pattern "Report_1_*.csv"
	files, err := filepath.Glob("Report*.csv")
	if err != nil {
		fmt.Println("Error getting files:", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No matching files found")
		return
	}

	// Process each file
	for _, file := range files {
		err := processFile(file)
		if err != nil {
			fmt.Printf("Error processing file %s: %v\n", file, err)
		}
	}

	// Output the overall total payment
	fmt.Printf("Total a pagar para todos los archivos: $%.2f\n", overallTotalPayment)
}

func processFile(filename string) error {
	// Open the CSV file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Parse the CSV file with pipe as delimiter
	reader := csv.NewReader(file)
	reader.Comma = '|' // Set the delimiter to pipe
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}

	personWorkData := make(map[string]int)        // Map to store total work minutes per person
	personPaymentData := make(map[string]float64) // Map to store total payment per person

	// Parse records
	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {
			if i == 0 {
				continue
			}
			vendor := row[0]
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Println("Error parsing total time:", err)
				continue
			}

			// Subtract 60 minutes for lunch break
			totalWorkDayMinutes -= 60

			// Calculate payment for the day
			payment := calculatePayment(totalWorkDayMinutes, vendor)

			// Output individual record result
			fmt.Printf("Archivo: %s, Nombre: %s, Fecha: %s, Pago: $%.2f\n", filename, vendor, entryTime.Format("2006-01-02"), payment)

			// Aggregate total work time and payment per person
			personWorkData[vendor] += totalWorkDayMinutes
			personPaymentData[vendor] += payment
		}
	}

	// Output total work time and payment for each person
	for vendor, totalWorkMinutes := range personWorkData {
		totalPayment := personPaymentData[vendor]
		totalHours := totalWorkMinutes / 60
		totalMinutes := totalWorkMinutes % 60
		fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado: %dh %dm, Total a pagar: $%.2f\n", filename, vendor, totalHours, totalMinutes, totalPayment)

		// Add to overall total payment
		overallTotalPayment += totalPayment
	}

	return nil
}

func parseTotalTimeToMinutes(total string) (int, error) {
	parts := strings.Split(total, " ")

	var hours, minutes int
	var err error

	if len(parts) >= 1 {
		if strings.Contains(parts[0], "h") {
			hours, err = strconv.Atoi(strings.TrimSuffix(parts[0], "h"))
			if err != nil {
				return 0, err
			}
		}
	}

	if len(parts) == 2 {
		if strings.Contains(parts[1], "m") {
			minutes, err = strconv.Atoi(strings.TrimSuffix(parts[1], "m"))
			if err != nil {
				return 0, err
			}
		}
	}

	return hours*60 + minutes, nil
}

func calculatePayment(totalWorkMinutes int, vendor string) float64 {
	hourlyPay := 1500.0 / 60
	extraTimePay := 1.5 * hourlyPay
	if vendor == "Dania Hidalgo" {
		hourlyPay = 1500.0 / 60 // Hourly pay converted to pay per minute
	}
	if vendor == "Josue Urena" {
		hourlyPay = 2300.0 / 60 // Hourly pay converted to pay per minute
	}
	if vendor == "Marjorie" {
		extraTimePay = hourlyPay
	}

	// Calculate payment
	if totalWorkMinutes > 8*60 {
		// Calculate extra time payment
		extraMinutes := totalWorkMinutes - 8*60
		return 8*60*hourlyPay + float64(extraMinutes)*extraTimePay
	}
	return float64(totalWorkMinutes) * hourlyPay
}
