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
	//colaborador string
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

	totalTiempoLaboradoAll := 0
	// Parse records
	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {
			if i == 0 {
				continue
			}
			colaborador := row[0]
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			//fmt.Println("entryTime:", entryTime)

			formattedDate := entryTime.Format("2006-01-02")
			// Print the formatted date
			//fmt.Println("Date:", formattedDate)

			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Println("Error parsing total time:", err)
				continue
			}

			// Subtract 60 minutes for lunch break
			//if colaborador != "Nayiry" {
			totalWorkDayMinutes -= 60
			//}
			isHoliday := 0
			if formattedDate == "2024-08-99" {
				isHoliday = 1
			}

			// Calculate payment for the day
			payment := calculatePayment(totalWorkDayMinutes, colaborador, isHoliday)

			// Output individual record result
			fmt.Printf("Archivo: %s, Nombre: %s, Fecha: %s, Pago: $%.2f, Holiday: $%b\n", filename, colaborador, entryTime.Format("2006-01-02"), payment, isHoliday)

			// Aggregate total work time and payment per person
			personWorkData[colaborador] += totalWorkDayMinutes
			personPaymentData[colaborador] += payment
		}
	}

	// Output total work time and payment for each person

	for colaborador, totalWorkMinutes := range personWorkData {
		totalPayment := personPaymentData[colaborador]
		totalHours := totalWorkMinutes / 60
		totalMinutes := totalWorkMinutes % 60
		rebajo := 0.0
		fmt.Println("------------------------------------------------------------------------")

		if colaborador == "Magally Loaiza" {
			totalPayment += (10000 + 26000)
			fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado + pasajes + 10 porc: %dh %dm, Total a pagar: $%.2f\n", filename, colaborador, totalHours, totalMinutes, totalPayment)
		}
		if colaborador == "Dania Hidalgo" {
			rebajo = 5000
			totalPayment += 20000
			totalPayment -= rebajo

			fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado + 10 por: %dh %dm,  CCSS:%.2f Total a pagar: $%.2f\n", filename, colaborador, totalHours, totalMinutes, rebajo, totalPayment)
		}
		if colaborador == "Nayiry" {
			rebajo = 50265
			totalPayment += 21000
			totalPayment -= rebajo

			fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado + 10 por: %dh %dm,  CCSS:%.2f Total a pagar: $%.2f\n", filename, colaborador, totalHours, totalMinutes, rebajo, totalPayment)
		}
		if colaborador == "Génesis" {
			totalPayment += 9000

			fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado + 10 porc: %dh %dm, Total a pagar: $%.2f\n", filename, colaborador, totalHours, totalMinutes, totalPayment)
		}
		if colaborador == "Vero" {
			totalPayment += 11000

			fmt.Printf("Archivo: %s, Nombre: %s, Total tiempo laborado + 10 porc: %dh %dm, Total a pagar: $%.2f\n", filename, colaborador, totalHours, totalMinutes, totalPayment)
		}
		totalTiempoLaboradoAll += totalHours

		// Add to overall total payment
		overallTotalPayment += totalPayment
		//fmt.Printf("Porcentaje Tiempo Laborado %dh\n", porc)
	}

	//porc := (totalHours * 100) / totalTiempoLaboradoAll

	fmt.Printf("Total Tiempo Laborado All %dh\n", totalTiempoLaboradoAll)

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

func calculatePayment(totalWorkMinutes int, colaborador string, isHoliday int) float64 {
	rate := 0.0
	hourlyPay := 0.0
	if colaborador == "Magally Loaiza" {
		rate = 1600
	}
	if colaborador == "Dania Hidalgo" {
		rate = 1600
	}
	if colaborador == "Nayiry" {
		rate = 3125
	}
	if colaborador == "Génesis" {
		rate = 1600
	}
	if colaborador == "Génesis" {
		rate = 1300
	}
	hourlyPay = rate / 60
	extraTimePay := hourlyPay
	if colaborador == "Magally Loaiza" {
		extraTimePay = 1.5 * hourlyPay
	}

	if isHoliday == 1 {
		hourlyPay = (rate * 2) / 60
		extraTimePay = 1.5 * hourlyPay
	}

	// Calculate payment
	if totalWorkMinutes > 8*60 {
		// Calculate extra time payment
		extraMinutes := totalWorkMinutes - 8*60
		//fmt.Println("Extra min", "colaborador", colaborador, extraMinutes)

		return 8*60*hourlyPay + float64(extraMinutes)*extraTimePay
	}
	return float64(totalWorkMinutes) * hourlyPay
}
