package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
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

func (d daychart) toHtmlRow() string {

    result := "<tbody>"
    subtotal := 0

    for _, intk := range d.intakes {
        result += "<tr>"
        result += intk.toHtmlRow()
        subtotal += intk.amount
        result += fmt.Sprintf("<td>%d</td>", subtotal)
        result += "</tr>"
    }

    result += "</tbody>"
    return result
}


func main() {
    
    intake1 := intake{
        time.Now(),
        100,
        "Lunch",
    }
    intake2 := intake{
        time.Now(),
        50,
        "Tablet",
    }

    today := daychart {
        []intake{intake1, intake2},
        1300,
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `
            <html>
                <head>
                    <title>
                        Track your water intake
                    </title>
                    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@4.4.1/dist/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
                </head>
                <body>
                    <div class="container">
                        <div >
                            <div class="pill"> Goal: %d ml </div>
                            <div class="pill"> Remaining: %d ml</div>
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
                        </div>
                    </div>
                </body>
            </html>
        `, today.goal, today.getRemaining(), today.toHtmlRow())
    })


    if err := http.ListenAndServe(":8000", nil ); err != nil {
        log.Fatal("Couldn't start server")
    }
}
