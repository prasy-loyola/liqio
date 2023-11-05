package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
	t "time"
)

type ioevent struct {
	id          int
	time        t.Time
	amount      int
	description string
}

type daychart struct {
	date    t.Time
	intakes []ioevent
	goal    int
}

var _dayFormat = "02012006"
var _timeFormat = "03:04 pm"
var _titleDateFormat = "2 Jan 2006"

var indiaLoc, _ = t.LoadLocation("Asia/Kolkata")

func (i ioevent) toHtmlRow(iotype string) string {

	return fmt.Sprintf(`
        <td>%s</td> 
        <td>%d</td> 
        <td>%s</td> 
        <td>
            <form hx-delete="/%s?date=%s&id=%d">
                <button type="submit">Delete</button>
            </form>
        </td>
        `,
		i.time.In(indiaLoc).Format(_timeFormat),
		i.amount,
		i.description,
        iotype,
		i.time.In(indiaLoc).Format(_dayFormat),
		i.id,
	)
}

func mapRowsToIOEvent(q *sql.Rows) []ioevent {

	events := []ioevent{}
	for q.Next() {
		var id int
		var day string
		var time t.Time
		var amount int
		var description string

		q.Scan(&id, &day, &time, &amount, &description)
		events = append(events, ioevent{
			id,
			time,
			amount,
			description,
		})
	}

	return events
}

func (d daychart) getRemaining() int {
	remaining := d.goal
	for _, intk := range d.intakes {
		remaining -= intk.amount
	}
	return remaining
}

func (d daychart) getTotal() int {
	total := 0
	for _, intk := range d.intakes {
		total += intk.amount
	}
	return total
}
func (d daychart) toHtml(iotype string) string {

	table := "<tbody>"
	subtotal := 0

	for _, intk := range d.intakes {
		table += "<tr>"
		table += intk.toHtmlRow(iotype)
		subtotal += intk.amount
		table += fmt.Sprintf("<td>%d</td>", subtotal)
		table += "</tr>"
	}

	table += "</tbody>"

    metadata := fmt.Sprintf(` <h4> Goal: <span class="badge badge-primary">%d ml </span></h4>
                            <h4> Remaining: <span class="badge badge-primary">%d ml </span></h4>
    `, d.goal, d.getRemaining() )
    if iotype == "output" {
        metadata = fmt.Sprintf(`<h4> Today's output: <span class="badge badge-primary">%d ml </span></h4> `, d.getTotal() )

    } 
    result := fmt.Sprintf(`<div name="data" id="data" >
                            %s
                            <form hx-post="/%s?date=%s">
                                <input type="text" value="%s" name="daypart" hidden/>
                                <label for="time">Time:</label><input type="time" value="" name="time"/>
                                <label for="amount">Amount:</label><input type="number" value="50" name="amount"/>
                                <label for="description">Description:</label><input type="text" name="description"/>
                                <button type="submit">Add</button>
                            </form>
                            <table class="table table-striped">
                                <thead>
                                    <tr>
                                        <th>Time</th>
                                        <th>Amount (ml)</th>
                                        <th>Description</th>
                                        <th>Delete</th>
                                        <th>Subtotal (ml)</th>
                                    </tr>
                                </thead>
                                %s
                            </table>
                        </div>`, metadata, iotype, d.date.Format(_dayFormat), d.date.Format(_dayFormat), table)
	return result
}

func handleIndexPageRequest(w http.ResponseWriter, r *http.Request) {

	dateParam := r.URL.Query().Get("date")
	date, err := t.Parse(_dayFormat, dateParam)

	if err != nil {
		date = t.Now()
	}

	fmt.Fprintf(w, `
            <html>
                <head>
                  <head>
                    <meta charset="utf-8">
                    <meta http-equiv="X-UA-Compatible" content="IE=edge">
                    <meta name="viewport" content="width=device-width, initial-scale=1">
                    <title>
                        Track your water intake
                    </title>
                    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@4.4.1/dist/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
                      <script src="https://unpkg.com/htmx.org@1.9.6"></script>
                </head>
                <body>
                    <div class="container">
                        <h2>Liquid Input/Output on %s</h2>
                        <div><h3>Water Intake</h3></div>
                        <div hx-get="/intake?date=%s" hx-trigger="intakeUpdate, load">Fetching data...</div>
                    </div>
                    <div class="container">
                        <div><h3>Urine Output</h3></div>
                        <div hx-get="/output?date=%s" hx-trigger="outputUpdate, load">Fetching data...</div>
                    </div>
                </body>
            </html>
        `, date.Format(_titleDateFormat), date.Format(_dayFormat), date.Format(_dayFormat))
}

func handleIORequest(iotype string, db *sql.DB, w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	date, err := t.Parse(_dayFormat, dateParam)

	if err != nil {
		http.Error(w, "Invalid Date", 400)
		return
	}
	if !(r.Method == "POST" || r.Method == "GET" || r.Method == "DELETE") {
		http.Error(w, "Invalid Request", 400)
		return
	}
	if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		idVal, err := strconv.Atoi(id)
		if err != nil {
			log.Print("id should be a number ", err)
			http.Error(w, "id should be a number", 400)
			return
		}
		stmt, err := db.Prepare("delete from " + iotype + " where ROWID = ? and day = ?;")
		defer stmt.Close()

		if err != nil {
			log.Print("Couldn't create prepared statements", err)
			return
		}

		_, err = stmt.Exec(idVal, dateParam)
		if err != nil {
			log.Printf("Couldn't delete %s record. \n %s", iotype, err)
			return
		}

		w.Header().Set("HX-Trigger", iotype+"Update")
		w.WriteHeader(200)
		return
	}

	if r.Method == "POST" {
		time := r.PostFormValue("time")
		amount := r.PostFormValue("amount")
		description := r.PostFormValue("description")
		amountVal, err := strconv.Atoi(amount)
		if err != nil {
			log.Print("Amount is in invalid format")
			http.Error(w, "Amount is in invalid format", 400)
			return
		}

		timeVal := t.Now()
		if time != "" {
			timeVal, err = t.ParseInLocation(_dayFormat+"15:04", dateParam+time, indiaLoc)
			if err != nil {
				log.Print("Time is in invalid format", err)
				http.Error(w, "Time is in invalid format", 400)
				return
			}
		}

		newIntake := ioevent{
			0, // dummy intake id which will not be stored in the db
			timeVal.UTC(),
			amountVal,
			description,
		}
		newIntake.insertIntoDb(iotype, dateParam, db)
		w.Header().Set("HX-Trigger", iotype+"Update")
		w.WriteHeader(200)
		return
	}
	intakes, err := queryIntakes(iotype, dateParam, db)

	chart := daychart{
		date,
		intakes,
		1300,
	}

	fmt.Fprint(w, chart.toHtml(iotype))
}

func handleOutputRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	log.Print("ERROR: no implemented")
	http.Error(w, "ERROR: Output not implemented", 500)
}

func setupTable(db *sql.DB) error {
	createTableStmt := `
        create table IF NOT EXISTS intake (day text, time TIMESTAMP default CURRENT_TIMESTAMP, amount number, description text);
        create table IF NOT EXISTS output (day text, time TIMESTAMP default CURRENT_TIMESTAMP, amount number, description text);
    `
	_, err := db.Exec(createTableStmt)
	if err != nil {
		log.Fatal("Couldn't create table or insert value", err)
	}
	return err
}

func (i ioevent) insertIntoDb(iotype string, date string, db *sql.DB) {
	stmt, err := db.Prepare("insert into " + iotype + " (day, time, amount, description) values (?, ?, ?, ?);")
	defer stmt.Close()

	if err != nil {
		log.Print("Couldn't create prepared statements", err)
	}

	_, err = stmt.Exec(date, i.time, i.amount, i.description)
	if err != nil {
		log.Printf("Couldn't insert %s record\n %s", iotype, err)
	}
}

func queryIntakes(iotype string, date string, db *sql.DB) ([]ioevent, error) {
	stmt, err := db.Prepare("select ROWID, day, time, amount, description from " + iotype + " where day = ? order by time;")
	if err != nil {
		log.Printf("Couldn't query %s records \n %s", iotype, err)
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(date)
	if err != nil {
		log.Printf("Couldn't query %s records \n %s", iotype, err)
		return nil, err
	}

	defer rows.Close()
	return mapRowsToIOEvent(rows), nil
}

func main() {

	db, err := sql.Open("sqlite3", "./intake.db")
	if err != nil {
		log.Fatal("Couldn't open database")
	}
	setupTable(db)
	defer db.Close()

	http.HandleFunc("/intake", func(w http.ResponseWriter, r *http.Request) {
		handleIORequest("intake", db, w, r)
	})
	http.HandleFunc("/output", func(w http.ResponseWriter, r *http.Request) {
		handleIORequest("output", db, w, r)
	})
	http.HandleFunc("/", handleIndexPageRequest)

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("Couldn't start server", err)
	}
}
