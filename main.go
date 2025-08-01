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
	EntryTime time.Time
	ExitTime  time.Time
	TotalWork time.Duration
}

type Employee struct {
	Name string
	Rate float64
	CCSS float64
}

// Define employees with their rates and CCSS deductions
var employees = []Employee{
	{Name: "Dani", Rate: 1800, CCSS: 0},
	{Name: "Nayi", Rate: 3125, CCSS: 0},
	{Name: "Vero", Rate: 1300, CCSS: 0},
	{Name: "Leidy", Rate: 2000, CCSS: 0},
	{Name: "Jose", Rate: 2000, CCSS: 0},
	{Name: "Graciela", Rate: 1800, CCSS: 0},
	{Name: "Ana", Rate: 1800, CCSS: 0},
	{Name: "Tatiana", Rate: 1800, CCSS: 0},
	{Name: "Angelica", Rate: 1800, CCSS: 0},
}

// Helper maps for quick lookup
var employeeRates = make(map[string]float64)
var ccssDeductions = make(map[string]float64)

func init() {
	for _, emp := range employees {
		employeeRates[emp.Name] = emp.Rate
		ccssDeductions[emp.Name] = emp.CCSS
	}
}

var overallTotalPayment float64 // Global variable to track total payment across all files

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <monto_servicio>")
		return
	}

	serviceAmountStr := os.Args[1]
	serviceAmount, err := strconv.ParseFloat(serviceAmountStr, 64)
	if err != nil {
		fmt.Println("Error: el monto de servicio debe ser un número")
		return
	}

	files, err := filepath.Glob("Report*.csv")
	if err != nil {
		fmt.Println("Error getting files:", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No matching files found")
		return
	}

	for _, file := range files {
		err := processFile(file, serviceAmount)
		if err != nil {
			fmt.Printf("Error processing file %s: %v\n", file, err)
		}
	}

	fmt.Printf("Total a pagar para todos los archivos: $%.2f\n", overallTotalPayment)
}

func processFile(filename string, serviceAmount float64) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '|'
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}

	personWorkData := make(map[string]int)
	personPaymentData := make(map[string]float64)

	totalTiempoLaboradoAll := 0

	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {
			if i == 0 {
				continue
			}
			colaborador := row[0]
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			formattedDate := entryTime.Format("2006-01-02")
			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Println("Error parsing total time:", err)
				continue
			}

			isHoliday := 0
			if formattedDate == "2025-07-25" {
				isHoliday = 1
			}

			payment := calculatePayment(totalWorkDayMinutes, colaborador, isHoliday)

			fmt.Printf("Archivo: %s, Nombre: %s, Fecha: %s, Pago: $%.2f, Holiday: %b\n",
				filename, colaborador, entryTime.Format("2006-01-02"), payment, isHoliday)

			personWorkData[colaborador] += totalWorkDayMinutes
			personPaymentData[colaborador] += payment
		}
	}

	totalMinutesWorkedAll := 0
	for _, minutes := range personWorkData {
		totalMinutesWorkedAll += minutes
	}

	for colaborador, totalWorkMinutes := range personWorkData {
		basePayment := personPaymentData[colaborador]
		totalHours := totalWorkMinutes / 60
		totalMinutes := totalWorkMinutes % 60

		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(totalWorkMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}

		ccss := ccssDeductions[colaborador]

		fmt.Println("------------------------------------------------------------------------")
		fmt.Printf("Archivo: %s, Nombre: %s\n", filename, colaborador)
		fmt.Printf("Total tiempo laborado: %dh %dm\n", totalHours, totalMinutes)
		fmt.Printf("Subtotal: $%.2f\n", basePayment)
		fmt.Printf("+ Servicio: $%.2f\n", proportionalService)
		fmt.Printf("- CCSS: $%.2f\n", ccss)

		totalPayment := basePayment + proportionalService - ccss
		fmt.Printf("Total a pagar: $%.2f\n", totalPayment)

		overallTotalPayment += totalPayment
		totalTiempoLaboradoAll += totalHours
	}

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

	fmt.Printf("Parsing Time: %s = %d hours, %d minutes\n", total, hours, minutes)
	return hours*60 + minutes, nil
}

func calculatePayment(totalWorkMinutes int, colaborador string, isHoliday int) float64 {
	rate := employeeRates[colaborador]
	hourlyPay := rate / 60
	extraTimePay := hourlyPay

	if isHoliday == 1 {
		hourlyPay = (rate * 2) / 60
		extraTimePay = 1.5 * hourlyPay
	}

	if totalWorkMinutes > 8*60 {
		extraMinutes := totalWorkMinutes - 8*60
		return 8*60*hourlyPay + float64(extraMinutes)*extraTimePay
	}
	return float64(totalWorkMinutes) * hourlyPay
}
