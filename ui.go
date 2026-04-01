package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func createApp() {
	a := app.New()
	w := a.NewWindow("BuruCafeteria - Sistema de Planilla")

	var selectedFiles []string

	// --- CSV file selection ---
	fileListLabel := widget.NewLabel("No hay archivos seleccionados")
	fileListLabel.Wrapping = fyne.TextWrapWord

	addFileBtn := widget.NewButton("Agregar CSV", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			path := reader.URI().Path()
			// Windows URI paths may start with /C:
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			selectedFiles = append(selectedFiles, path)
			fileListLabel.SetText(strings.Join(selectedFiles, "\n"))
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
		fd.Show()
	})

	clearFilesBtn := widget.NewButton("Limpiar", func() {
		selectedFiles = nil
		fileListLabel.SetText("No hay archivos seleccionados")
	})

	fileSection := container.NewVBox(
		widget.NewLabelWithStyle("Archivos CSV", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(addFileBtn, clearFilesBtn),
		fileListLabel,
	)

	// --- Service amount ---
	serviceEntry := widget.NewEntry()
	serviceEntry.SetPlaceHolder("Ej: 50000")

	serviceSection := container.NewVBox(
		widget.NewLabelWithStyle("Monto de servicio a repartir", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		serviceEntry,
	)

	// --- Holidays ---
	holidayEntry := widget.NewEntry()
	holidayEntry.SetPlaceHolder("Ej: 2025-07-25, 2025-08-02")

	holidaySection := container.NewVBox(
		widget.NewLabelWithStyle("Días feriados (YYYY-MM-DD, separados por coma)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		holidayEntry,
	)

	// --- Employee table ---
	type empRow struct {
		ccssEntry *widget.Entry
		vacEntry  *widget.Entry
	}
	empRows := make([]empRow, len(employees))

	empGrid := container.NewVBox()

	// Header
	empGrid.Add(container.NewGridWithColumns(4,
		widget.NewLabelWithStyle("Nombre", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Tarifa/h", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Rebajo CCSS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Días Vacaciones", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	))

	for i, emp := range employees {
		ccssE := widget.NewEntry()
		ccssE.SetText(fmt.Sprintf("%.2f", emp.CCSS))

		vacE := widget.NewEntry()
		vacE.SetText(fmt.Sprintf("%d", emp.VacationDays))

		empRows[i] = empRow{ccssEntry: ccssE, vacEntry: vacE}

		empGrid.Add(container.NewGridWithColumns(4,
			widget.NewLabel(emp.Name),
			widget.NewLabel(fmt.Sprintf("₡%.0f", emp.Rate)),
			ccssE,
			vacE,
		))
	}

	empSection := container.NewVBox(
		widget.NewLabelWithStyle("Empleados", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		empGrid,
	)

	// --- Results ---
	resultsText := widget.NewMultiLineEntry()
	resultsText.SetPlaceHolder("Los resultados aparecerán aquí después de calcular...")
	resultsText.Wrapping = fyne.TextWrapOff

	resultsSection := container.NewVBox(
		widget.NewLabelWithStyle("Resultados", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		resultsText,
	)

	// --- Calculate button ---
	calcBtn := widget.NewButton("Calcular Planilla", func() {
		// Validate files
		if len(selectedFiles) == 0 {
			dialog.ShowError(fmt.Errorf("debe seleccionar al menos un archivo CSV"), w)
			return
		}

		// Parse service amount
		serviceAmount, err := strconv.ParseFloat(strings.TrimSpace(serviceEntry.Text), 64)
		if err != nil {
			dialog.ShowError(fmt.Errorf("el monto de servicio debe ser un número válido"), w)
			return
		}

		// Parse holidays
		var holidays []string
		if trimmed := strings.TrimSpace(holidayEntry.Text); trimmed != "" {
			for _, h := range strings.Split(trimmed, ",") {
				if h = strings.TrimSpace(h); h != "" {
					holidays = append(holidays, h)
				}
			}
		}

		// Update employees from UI inputs
		for i := range employees {
			if val, err := strconv.ParseFloat(strings.TrimSpace(empRows[i].ccssEntry.Text), 64); err == nil {
				employees[i].CCSS = val
			}
			if val, err := strconv.Atoi(strings.TrimSpace(empRows[i].vacEntry.Text)); err == nil {
				employees[i].VacationDays = val
			}
		}

		// Run payroll and display results
		result := runPayroll(selectedFiles, serviceAmount, holidays)
		resultsText.SetText(result)
	})
	calcBtn.Importance = widget.HighImportance

	// --- Main layout ---
	content := container.NewVBox(
		widget.NewLabelWithStyle("BuruCafeteria - Sistema de Planilla",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		fileSection,
		widget.NewSeparator(),
		serviceSection,
		widget.NewSeparator(),
		holidaySection,
		widget.NewSeparator(),
		empSection,
		widget.NewSeparator(),
		calcBtn,
		widget.NewSeparator(),
		resultsSection,
	)

	w.SetContent(container.NewScroll(content))
	w.Resize(fyne.NewSize(850, 700))
	w.ShowAndRun()
}
