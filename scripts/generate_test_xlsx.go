package main

import (
	"log"
	"strconv"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	sh := "Sheet1"
	f.NewSheet(sh)
	// Header row
	header := []interface{}{"Name", "Email", "Department", "Grade", "DateOfJoining", "DateOfLeaving"}
	if err := f.SetSheetRow(sh, "A1", &header); err != nil {
		log.Fatalf("set header: %v", err)
	}

	totalRows := 1000000
	for i := 1; i <= totalRows; i++ {
		row := []interface{}{
			"bob" + strconv.Itoa(i),
			"bob" + strconv.Itoa(i) + "@gmail.com",
			"Product",
			"P1",
			"2020-01-01",
			"",
		}
		cell := "A" + strconv.Itoa(i+1) // +1 because header is at row 1
		if err := f.SetSheetRow(sh, cell, &row); err != nil {
			log.Fatalf("set row %d: %v", i, err)
		}
	}
	if err := f.SaveAs("test_employees.xlsx"); err != nil {
		log.Fatalf("save: %v", err)
	}
	log.Println("wrote test_employees.xlsx")
}
