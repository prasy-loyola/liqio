package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	t "time"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

type intake struct {
    id          int
	time        t.Time
	amount      int
	description string
}

type daychart struct {
	date    t.Time
	intakes []intake
	goal    int
}

var indiaLoc, _ = t.LoadLocation("Asia/Kolkata")

func (i intake) toHtmlRow() string {

	return fmt.Sprintf(`
        <td>%s</td> 
        <td>%d</td> 
        <td>%s</td> 
        <td>
            <form hx-delete="/intake?date=%s&id=%d">
                <button type="submit">Delete</button>
            </form>
        </td>
        `,
		i.time.In(indiaLoc).Format("03:04 pm"),
		i.amount,
		i.description,
		i.time.In(indiaLoc).Format("02012006"),
        i.id,
    )
}

func mapRowsToIntake(q *sql.Rows) []intake {
    
    intakes := []intake{};
    for q.Next() {
        var id int
        var day string
        var time t.Time
        var amount int
        var description string

        q.Scan(&id, &day, &time, &amount, &description)
        intakes = append(intakes, intake{
            id,
            time,
            amount,
            description,
        })
    }

    return intakes
}

func (d daychart) getRemaining() int {
	remaining := d.goal
	for _, intk := range d.intakes {
		remaining -= intk.amount
	}
	return remaining
}

func (d daychart) toHtml() string {

	table := "<tbody>"
	subtotal := 0

	for _, intk := range d.intakes {
		table += "<tr>"
		table += intk.toHtmlRow()
		subtotal += intk.amount
		table += fmt.Sprintf("<td>%d</td>", subtotal)
		table += "</tr>"
	}

	table += "</tbody>"

	result := fmt.Sprintf(`<div name="data" id="data" >
                            <h4> Goal: <span class="badge badge-primary">%d ml </span></h4>
                            <h4> Remaining: <span class="badge badge-primary">%d ml </span></h4>
                            <form hx-post="/intake?date=%s" hx-target="#data" hx-swap>
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
                        </div>`, d.goal, d.getRemaining(), d.date.Format("02012006"), table)
	return result
}

func handleIndexPageRequest(w http.ResponseWriter, r *http.Request) {

	dateParam := r.URL.Query().Get("date")
	date, err := t.Parse("02012006", dateParam)

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
                        <div><h2>Water Intake on %s</h2></div>
                        <div hx-get="/intake?date=%s" hx-trigger="load, intakeUpdate from:body">Fetching data...</div>
                    </div>
                </body>
            </html>
        `, date.Format("2 Jan 2006"), date.Format("02012006"))
}

func handleIntakeRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	date, err := t.Parse("02012006", dateParam)

	if err != nil {
		http.Error(w, "Invalid Date", 400)
		return
	}
	if ! (r.Method == "POST" || r.Method == "GET" || r.Method == "DELETE") {
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
        stmt, err := db.Prepare("delete from intake where ROWID = ? and day = ?;")
        defer stmt.Close()

        if err != nil {
            log.Print("Couldn't create prepared statements", err)
            return
        }

        _, err = stmt.Exec(idVal, dateParam)
        if err != nil {
            log.Print("Couldn't delete intake record", err)
            return
        }

        w.Header().Set("HX-Trigger", "intakeUpdate")
        w.WriteHeader(200)
        return
    }

	if r.Method == "POST" {
		amount := r.PostFormValue("amount")
		description := r.PostFormValue("description")
		amountVal, err := strconv.Atoi(amount)
		if err != nil {
			log.Print("Amount is in invalid format")
            http.Error(w, "Amount is in invalid format", 400)
            return
		}

        if dateParam != t.Now().Format("02012006") {
			log.Print("Cannot edit/add intake into past days")
            http.Error(w, "Cannot edit/add intake into past days", 400)
            return
        }

		newIntake := intake{
            0, // dummy intake id which will not be stored in the db
			t.Now(),
			amountVal,
			description,
		}
        newIntake.insertIntoDb(dateParam, db)
	}
    intakes, err := queryIntakes(dateParam, db)

    chart := daychart{
        date, 
        intakes,
	    1300,
    }

	fmt.Fprint(w, chart.toHtml())
}


func main() {
    
    db, err := sql.Open("sqlite3", "./intake.db")
    if err != nil {
        log.Fatal("Couldn't open database")
    }
    //setupTable(db)
    defer db.Close()	

	http.HandleFunc("/intake", func(w http.ResponseWriter, r *http.Request) {
		handleIntakeRequest(db, w, r)
	})
	http.HandleFunc("/", handleIndexPageRequest)

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("Couldn't start server", err)
	}
}


func setupTable(db *sql.DB) error {
    createTableStmt := `
        create table intake (day text, time TIMESTAMP default CURRENT_TIMESTAMP, amount number, description text);
    `
    _, err := db.Exec(createTableStmt)
    if err != nil {
        log.Fatal("Couldn't create table or insert value")
    }
    return err
}


func (i intake) insertIntoDb(date string, db *sql.DB) {
    stmt, err := db.Prepare("insert into intake(day, amount, description) values (?, ?, ?);")
    defer stmt.Close()

    if err != nil {
        log.Fatal("Couldn't create prepared statements", err)
    }

    _, err = stmt.Exec(date, i.amount, i.description)
    if err != nil {
        log.Fatal("Couldn't insert intake record", err)
    }
}

func queryIntakes(date string, db *sql.DB) ([]intake, error) {
    stmt, err := db.Prepare("select ROWID, day, time, amount, description from intake where day = ?;")
    if err != nil {
        log.Fatal("Couldn't query intake records")
        return nil, err
    }
    defer stmt.Close()
    rows, err := stmt.Query(date)
    if err != nil {
        log.Fatal("Couldn't query intake records")
        return nil, err
    }
    
    defer rows.Close()
    intakes := mapRowsToIntake(rows)
    return intakes, nil
}

