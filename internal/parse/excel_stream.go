package parse

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
	"handling_large_files_go/internal/db"
)

// ProcessXLSX streams the xlsx file, validates rows, performs batch upserts and writes invalid rows to a file.
func ProcessXLSX(path string) (invalidFilePath string, err error) {
	if db.DB == nil {
		if err := db.Connect(); err != nil {
			return "", err
		}
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// use first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("no sheets in file")
	}
	sheet := sheets[0]
	rows, err := f.Rows(sheet)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	rowCount := 0
	batch := make([][]interface{}, 0, 100)
	invalidRows := make([][]interface{}, 0)
	// read header
	if rows.Next() {
		headCols, _ := rows.Columns()
		_ = headCols
	}

	for rows.Next() {
		cols, err := rows.Columns()
		if err != nil {
			return "", err
		}
		rowCount++
		// basic validation: expect at least Name and Email in columns 1 and 2 (0-indexed)
		name := ""
		email := ""
		if len(cols) > 0 && cols[0] != "" {
			name = cols[0]
		}
		if len(cols) > 1 && cols[1] != "" {
			email = cols[1]
		}
		if name == "" || email == "" {
			// convert to interface slice and append error column
			iface := make([]interface{}, len(cols))
			for i := range cols {
				iface[i] = cols[i]
			}
			iface = append(iface, "missing name or email")
			invalidRows = append(invalidRows, iface)
			continue
		}
		// prepare upsert values: name,email,department,grade,date_of_joining,date_of_leaving
		vals := make([]interface{}, 6)
		vals[0] = name
		vals[1] = email
		if len(cols) > 2 {
			vals[2] = cols[2]
		}
		if len(cols) > 3 {
			vals[3] = cols[3]
		}
		// TODO: parse dates properly
		batch = append(batch, vals)
		if len(batch) >= 100 {
			if err := flushBatch(batch); err != nil {
				log.Println("batch upsert err:", err)
				return "", err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := flushBatch(batch); err != nil {
			return "", err
		}
	}

	// if invalidRows exist, write to a file
	if len(invalidRows) > 0 {
		out := os.TempDir() + "/invalid-" + fmt.Sprint(time.Now().Unix()) + ".xlsx"
		of := excelize.NewFile()
		sh := "Sheet1"
		of.NewSheet(sh)
		for i, r := range invalidRows {
			cells := make([]interface{}, len(r))
			for j := range r {
				cells[j] = r[j]
			}
			of.SetSheetRow(sh, fmt.Sprintf("A%d", i+1), &cells)
		}
		if err := of.SaveAs(out); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", nil
}

func flushBatch(batch [][]interface{}) error {
	// build SQL for batch upsert
	// INSERT INTO employees (name,email,department,grade,date_of_joining,date_of_leaving) VALUES ... ON CONFLICT (name,email) DO UPDATE SET department=EXCLUDED.department, grade=EXCLUDED.grade, date_of_joining=EXCLUDED.date_of_joining, date_of_leaving=EXCLUDED.date_of_leaving, updated_at=now();
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	stmt := "INSERT INTO employees (name,email,department,grade,date_of_joining,date_of_leaving) VALUES "
	params := []interface{}{}
	place := []string{}
	cnt := 1
	for _, row := range batch {
		place = append(place, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", cnt, cnt+1, cnt+2, cnt+3, cnt+4, cnt+5))
		cnt += 6
		for i := 0; i < 6; i++ {
			params = append(params, row[i])
		}
	}
	stmt += fmt.Sprintf("%s ON CONFLICT (name,email) DO UPDATE SET department=EXCLUDED.department, grade=EXCLUDED.grade, date_of_joining=EXCLUDED.date_of_joining, date_of_leaving=EXCLUDED.date_of_leaving, updated_at=now()", join(place, ","))
	if _, err := tx.Exec(stmt, params...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func join(a []string, sep string) string {
	if len(a) == 0 {
		return ""
	}
	res := a[0]
	for i := 1; i < len(a); i++ {
		res += sep + a[i]
	}
	return res
}
