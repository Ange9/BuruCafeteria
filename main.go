package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Name         string
	Rate         float64
	CCSS         float64
	VacationDays int // Add this field

}

// Define employees with their rates and CCSS deductions
var employees = []Employee{
	{Name: "Dani", Rate: 1800, CCSS: 10000, VacationDays: 9},
	{Name: "Nayi", Rate: 3125, CCSS: 10000, VacationDays: 0},
	{Name: "Vero", Rate: 1300, CCSS: 0, VacationDays: 0},
	{Name: "Leidy", Rate: 2000, CCSS: 0, VacationDays: 0},
	{Name: "Jose Mario", Rate: 2000, CCSS: 0, VacationDays: 0},
	{Name: "Graciela", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Ana", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Tatiana", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Angélica", Rate: 1800, CCSS: 0, VacationDays: 0},
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

// New: Map to store hours per worker per day
var hoursPerWorkerPerDay = make(map[string]map[string]float64)

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

	showBarGraphAndPayments(serviceAmount)
}

// New: Map to store break minutes per worker per day
var breaksPerWorkerPerDay = make(map[string]map[string]float64)

// For break calculation between sessions
type session struct {
	entry time.Time
	exit  time.Time
	mins  int
}

// Map: worker -> date -> []session
var sessionsPerWorkerPerDay = make(map[string]map[string][]session)

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
			colaborador := strings.TrimSpace(row[0])
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			exitTime, _ := time.Parse("2/1/2006 15:04", row[2])
			formattedDate := entryTime.Format("2006-01-02")
			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Println("Error parsing total time:", err)
				continue
			}

			// Store session for break calculation
			if _, ok := sessionsPerWorkerPerDay[colaborador]; !ok {
				sessionsPerWorkerPerDay[colaborador] = make(map[string][]session)
			}
			sessionsPerWorkerPerDay[colaborador][formattedDate] = append(
				sessionsPerWorkerPerDay[colaborador][formattedDate],
				session{entry: entryTime, exit: exitTime, mins: totalWorkDayMinutes},
			)

			// Store hours per worker per day
			if _, ok := hoursPerWorkerPerDay[colaborador]; !ok {
				hoursPerWorkerPerDay[colaborador] = make(map[string]float64)
			}
			hoursPerWorkerPerDay[colaborador][formattedDate] += float64(totalWorkDayMinutes) / 60.0

			isHoliday := 0
			if formattedDate == "2025-07-25" {
				isHoliday = 1
			}

			payment := calculatePayment(totalWorkDayMinutes, colaborador, isHoliday)

			// fmt.Printf("Archivo: %s, Nombre: %s, Fecha: %s, Pago: $%.2f, Holiday: %b\n",
			// 	filename, colaborador, entryTime.Format("2006-01-02"), payment, isHoliday)

			personWorkData[colaborador] += totalWorkDayMinutes
			personPaymentData[colaborador] += payment
		}
	}

	// Calculate breaks between sessions per worker per day
	for colaborador, days := range sessionsPerWorkerPerDay {
		if _, ok := breaksPerWorkerPerDay[colaborador]; !ok {
			breaksPerWorkerPerDay[colaborador] = make(map[string]float64)
		}
		for date, sessions := range days {
			// Sort sessions by entry time
			sort.Slice(sessions, func(i, j int) bool {
				return sessions[i].entry.Before(sessions[j].entry)
			})
			breakMinutes := 0.0
			for i := 1; i < len(sessions); i++ {
				// Break is time between previous exit and current entry
				breakMin := sessions[i].entry.Sub(sessions[i-1].exit).Minutes()
				if breakMin > 0 {
					breakMinutes += breakMin
				}
			}
			breaksPerWorkerPerDay[colaborador][date] = breakMinutes
		}
	}

	totalMinutesWorkedAll := 0
	for _, minutes := range personWorkData {
		totalMinutesWorkedAll += minutes
	}

	for colaborador, totalWorkMinutes := range personWorkData {
		basePayment := personPaymentData[colaborador]
		totalHours := totalWorkMinutes / 60
		// totalMinutes := totalWorkMinutes % 60

		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(totalWorkMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}

		ccss := ccssDeductions[colaborador]

		totalPayment := basePayment + proportionalService - ccss

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

	// fmt.Printf("Parsing Time: %s = %d hours, %d minutes\n", total, hours, minutes)
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

func showBarGraphAndPayments(serviceAmount float64) {
	fmt.Println("\nResumen de horas trabajadas, descansos y pagos por colaborador (ordenados):")
	fmt.Printf("Monto total de servicio a repartir: $%.2f\n", serviceAmount)
	fmt.Println("-------------------------------------------------------------------")

	// Get sorted list of employee names
	var employeeNames []string
	for name := range hoursPerWorkerPerDay {
		employeeNames = append(employeeNames, name)
	}
	sort.Strings(employeeNames)

	// Calculate total minutes worked by all employees (excluding vacation days)
	totalMinutesWorkedAll := 0
	for _, emp := range employeeNames {
		empObj := getEmployeeByName(emp)
		vacDays := 0
		if empObj != nil {
			vacDays = empObj.VacationDays
		}
		vacMinutes := vacDays * 8 * 60
		empMinutes := 0
		for _, mins := range hoursPerWorkerPerDay[emp] {
			empMinutes += int(mins * 60)
		}
		if empMinutes < vacMinutes {
			vacMinutes = empMinutes // Prevent negative
		}
		totalMinutesWorkedAll += empMinutes - vacMinutes
	}

	for _, colaborador := range employeeNames {
		fmt.Printf("%s:\n", colaborador)
		days := hoursPerWorkerPerDay[colaborador]

		// Sort the dates for this worker
		var dates []string
		for date := range days {
			dates = append(dates, date)
		}
		sort.Strings(dates)

		totalHours := 0.0
		totalBreak := 0.0
		totalMinutes := 0

		for _, date := range dates {
			hours := days[date]
			breakMin := breaksPerWorkerPerDay[colaborador][date]
			totalHours += hours
			totalBreak += breakMin
			totalMinutes += int(hours * 60)

			// Get sessions for this worker and date
			sessions := sessionsFor(colaborador, date)
			sessionStr := ""
			if len(sessions) > 0 {
				sessionStr += fmt.Sprintf("Entrada: %s | Salida: %s", sessions[0].entry.Format("03:04 PM"), sessions[0].exit.Format("03:04 PM"))
				for i := 1; i < len(sessions); i++ {
					breakDuration := sessions[i].entry.Sub(sessions[i-1].exit)
					breakMin := int(breakDuration.Minutes())
					if breakMin > 0 {
						sessionStr += fmt.Sprintf(" | Descanso: %d min", breakMin)
					}
					sessionStr += fmt.Sprintf(" | Entrada: %s | Salida: %s", sessions[i].entry.Format("03:04 PM"), sessions[i].exit.Format("03:04 PM"))
				}
			}

			fmt.Printf("  %s | %6.2f h  | Descanso total: %2.0f min | %s\n", date, hours, breakMin, sessionStr)
		}

		// Vacation calculation
		empObj := getEmployeeByName(colaborador)
		vacDays := 0
		if empObj != nil {
			vacDays = empObj.VacationDays
		}
		vacHours := float64(vacDays) * 8.0
		vacMinutes := vacDays * 8 * 60

		fmt.Printf("  TOTAL HORAS: %5.2f h | TOTAL DESCANSO: %.0f min\n", totalHours, totalBreak)
		if vacDays > 0 {
			fmt.Printf("  DÍAS DE VACACIONES: %d (%.2f h no cuentan para servicio)\n", vacDays, vacHours)
		}

		// Payment summary for this employee
		totalPayMinutes := totalMinutes + vacMinutes

		// Apply 1.5x rate for minutes after 96 hours (5760 min)
		normalMinutes := totalPayMinutes
		extraMinutes := 0
		if totalPayMinutes > 96*60 {
			normalMinutes = 96 * 60
			extraMinutes = totalPayMinutes - normalMinutes
		}

		// Convert minutes to hours for display
		normalHours := float64(normalMinutes) / 60.0
		extraHours := float64(extraMinutes) / 60.0

		rate := employeeRates[colaborador]
		normalPay := float64(normalMinutes) * rate / 60
		extraPay := float64(extraMinutes) * rate / 60 * 1.5
		basePayment := normalPay + extraPay

		// Split worked and vacation amounts for display
		workedPay := 0.0
		vacationPay := 0.0
		if totalPayMinutes > 0 {
			workedRatio := float64(totalMinutes) / float64(totalPayMinutes)
			vacationRatio := float64(vacMinutes) / float64(totalPayMinutes)
			workedPay = basePayment * workedRatio
			vacationPay = basePayment * vacationRatio
		}

		ccss := ccssDeductions[colaborador]

		// Service is only for worked minutes (not vacation)
		serviceMinutes := totalMinutes
		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(serviceMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}
		totalPayment := workedPay + vacationPay + proportionalService - ccss

		fmt.Printf("  Tiempo normal: %.2f h | Monto normal: $%.2f\n", normalHours, normalPay)
		fmt.Printf("  Tiempo extra:  %.2f h | Monto extra:  $%.2f\n", extraHours, extraPay)

		if vacDays > 0 {
			fmt.Printf("  Monto por días trabajados: $%.2f\n", workedPay)
			fmt.Printf("  Monto por vacaciones:      $%.2f\n", vacationPay)
		} else {
			fmt.Printf("  Monto por días trabajados: $%.2f\n", workedPay)
		}
		fmt.Printf("  Servicio: $%.2f | CCSS: $%.2f | TOTAL: $%.2f\n\n",
			proportionalService, ccss, totalPayment)
	}
}

func getEmployeeByName(name string) *Employee {
	for i := range employees {
		if employees[i].Name == name {
			return &employees[i]
		}
	}
	return nil
}

// Helper to get sessions for a worker and date
func sessionsFor(worker, date string) []struct{ entry, exit time.Time } {
	sessions := []struct{ entry, exit time.Time }{}
	if workerSessions, ok := sessionsPerWorkerPerDay[worker]; ok {
		if daySessions, ok := workerSessions[date]; ok {
			for _, s := range daySessions {
				sessions = append(sessions, struct{ entry, exit time.Time }{s.entry, s.exit})
			}
		}
	}
	return sessions
}
