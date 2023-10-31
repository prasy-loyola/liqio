package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	t "time"
)


type intake struct {
    time t.Time
    amount int
    description string
}

type daychart struct {
    intakes []intake
    goal int
}

func (i intake) toHtmlRow() string {

    return fmt.Sprintf( `
        <td>%s</td> 
        <td>%d</td> 
        <td>%s</td> 
        `, 
        i.time, 
        i.amount, 
        i.description)
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
                            <form hx-post="/intake" hx-target="#data" hx-swap>
                                <label for="amount">Amount:</label><input type="number" name="amount"/>
                                <label for="description">Description:</label><input type="text" name="description"/>
                                <button type="submit">Add</button>
                            </form>
                            <table class="table table-striped">
                                <thead>
                                    <tr>
                                        <th>Time</th>
                                        <th>Amount (ml)</th>
                                        <th>Description</th>
                                        <th>Subtotal (ml)</th>
                                    </tr>
                                </thead>
                                %s
                            </table>
                        </div>`, d.goal, d.getRemaining(), table)
    return result
}

func main() {
    
    today := daychart {
        []intake{},
        1300,
    }

    http.HandleFunc("/intake", func(w http.ResponseWriter, r *http.Request) {

        if r.Method == "POST" {
            amount := r.PostFormValue("amount")
            description := r.PostFormValue("description")
            amountVal, err := strconv.Atoi(amount);
            if err != nil {
                log.Fatal("Amount is in invalid format")
            } 

            newIntake := intake{
                t.Now(),
                amountVal,
                description,
            }
            today.intakes = append(today.intakes, newIntake)
        }

        fmt.Fprint(w, today.toHtml());
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `
            <html>
                <head>
                    <title>
                        Track your water intake
                    </title>
                    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@4.4.1/dist/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
                      <script src="https://unpkg.com/htmx.org@1.9.6"></script>
                </head>
                <body>
                    <div class="container">
                        <div hx-get="/intake" hx-trigger="load">Fetching data...</div>
                    </div>
                </body>
            </html>
        `)
    })


    if err := http.ListenAndServe(":8000", nil ); err != nil {
        log.Fatal("Couldn't start server")
    }
}
