* Shigoto: Very Simple Job Managing System
wip

** How to Use

curl http://***/postjob --include  --header "Content-Type: application/json" --request "POST"  --data '{"output": "path to logfile","command": "command"}'
curl http://***/getjob --include --header "Content-Type: application/json" --request "GET"

curl http://***/notifystate --include --header "Content-Type: application/json" --request "POST" --data '{"ID" 0,"state": finished}'
curl http://***/getqueue --include --header "Content-Type: application/json" --request "GET"
