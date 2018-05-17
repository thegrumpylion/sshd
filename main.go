package main

import "log"

func main() {
	err := SSHdListenAndServe(":2222", []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDKqTdqPnDiOMCZ6tIjz8RbwWBZ6E92HTewH+C39brX4Fi6EKsEOBFNNoiwx05w9dAJLdmjHPd7noLO5zCClYIum6QYakq6nk9TBrIa+PsTq/GvYw5W/Ga/lbqXHfNr4CEfvoSrfbH3+5AHIgpFDGNRTlvUSyKG2st1ekWqR3LzaqAIDo6JvWmAbvmN9yCkF7iTQTQC35B4l0J23+kiAlumc/PTRUfcoTAzKdiPUlOythY6NzNXGHJF5dSJWmxmICF6BAqpWYSDeG+k+CAHwFNPs7Xe3knF3STQ+shxK/48JL9b5C+rmVNqiC+vBwL71VNBoxIJswCDJPPqssbCw76F nikolas.sepos@gmail.com"})
	if err != nil {
		log.Fatalf("%v", err)
	}
}
