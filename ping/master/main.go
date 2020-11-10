package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
)

type event struct {
	Name    string `json:"name"`
	Rps     uint64 `json:"rps"`
	Latency uint64 `json:"latency"`
	Errors  uint64 `json:"errors"`
}

var (
	addr = flag.String("addr", "localhost:8080", "http service address")
	// TODO
	// token     = flag.String("token", "", "dashboard access token")
	// enableTerminal  = flag.Bool("terminal", true, "enable terminal dashboard")
	enableDashboard = flag.Bool("dashboard", true, "enable web dashboard")
	upgrader        = websocket.Upgrader{}
	events          = make(chan event)
)

func dashboard(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %s\n", err)
		return
	}
	defer c.Close()

	for e := range events {
		bytes, err := json.Marshal(e)
		if err != nil {
			log.Printf("failed to marshal event data: %s\n", err)
			continue
		}
		c.WriteMessage(websocket.TextMessage, bytes)
	}
}

func eventListener(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %s\n", err)
		return
	}
	defer c.Close()
	for {
		// TODO: check message type
		mt, bytes, err := c.ReadMessage()
		if err != nil {
			log.Printf("failed to read message: %s\n", err)
			break
		}
		if mt != websocket.TextMessage {
			log.Printf("unknown ws message type: %d\n", mt)
			break
		}
		var message event
		err = json.Unmarshal(bytes, &message)
		if err != nil {
			log.Printf("failed to unmarshal event data: %s\n", err)
			continue
		}
		message.Name = fmt.Sprintf("%s (%s)", message.Name, r.RemoteAddr)
		events <- message
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	dashboardTemplate.Execute(w, "ws://"+r.Host+"/dashboard")
}

func main() {
	flag.Parse()
	http.HandleFunc("/events", eventListener)
	if *enableDashboard {
		http.HandleFunc("/dashboard", dashboard)
		http.HandleFunc("/", index)
	}
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var dashboardTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto:300,300italic,700,700italic">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/milligram/1.4.1/milligram.css">
    <title>Dashboard</title>
	<script> 
        window.addEventListener("load", function(evt) {
			var workers = [];
            var ws = new WebSocket("{{.}}");
            ws.onclose = function(event) {
                location.reload();
			}
			ws.onmessage = function(event) {
				var exists = false;
				var data = JSON.parse(event.data);
				workers.forEach(function(worker, index) {
					if (worker.name === data.name) {
						data.updated = Date.now();
						workers[index] = data;

						exists = true;
					}
				});
				if (!exists) {
					workers.push(data)
				}
			}
			window.setInterval(function(){
				var container = document.getElementById("container");
				var totalRps = 0;
				var avgLatency = 0;
				var totalErrors = 0;
				workers.forEach(function(worker, index) {
					if (Math.floor((Date.now() - worker.updated)/1000) > 4) {
						var workerNode = document.getElementById(worker.name);
						workerNode.remove();
						workers.splice(index, 1);
					} else {
						totalRps += worker.rps;
						avgLatency = (avgLatency+worker.latency)/2;
						totalErrors += worker.errors;
						var name = worker.name;
						var workerNode = document.getElementById(worker.name);
						if (workerNode === null || workerNode === undefined) {
							workerNode = document.createElement("div");
							workerNode.setAttribute("id", worker.name);
							var ul = document.createElement("ul");
							ul.setAttribute("id", "list");
							var header = document.createElement("h1")
							var rps = document.createElement("li");
							var latency = document.createElement("li");
							var errors = document.createElement("li");
							header.setAttribute("id", name+"-name")
							rps.setAttribute("id", name+"-rps");
							latency.setAttribute("id", name+"-latency");
							errors.setAttribute("id", name+"-errors");
							ul.appendChild(rps);
							ul.appendChild(latency);
							ul.appendChild(errors);
							workerNode.appendChild(header);
							workerNode.appendChild(ul);
							container.appendChild(workerNode);
						}
						document.getElementById(name+"-name").innerHTML = worker.name;
						document.getElementById(name+"-rps").innerHTML = "RPS: "+worker.rps;
						document.getElementById(name+"-latency").innerHTML = "Latency: "+worker.latency;
						document.getElementById(name+"-errors").innerHTML = "Errors: "+worker.errors;	
					}
				});
				document.getElementById("totalRps").innerHTML = "RPS: "+totalRps;
				document.getElementById("avgLatency").innerHTML = "Latency: "+avgLatency;
				document.getElementById("totalErrors").innerHTML = "Total errors: "+totalErrors;
			  }, 1000);
			
        });
    </script>
</head>
<body>
	<div style="width: 40%;margin-top: 5em;" id="container" class="container">
		<div id="total">
			<h1>Total</h1>
			<ul>
				<li id="totalRps">Rps: </li>
				<li id="avgLatency">Latency: </li>
				<li id="totalErrors">Total errors: </li>
			</ul>
			<hr/>
		</div>
    </div>
</body>
</html>
`))
