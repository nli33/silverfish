#!/usr/bin/expect -f

set timeout -1

spawn go run ./cmd/silverfish --profile

send "uci\r"
send "position startpos\r"
send "go perft depth 5\r"

expect {
    -re {Perft result:} {
        send "quit\r"
    }
}

expect eof
