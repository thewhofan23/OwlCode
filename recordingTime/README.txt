Hello!

This project was started to assist support with calculating total recording time using graphQL. It’s intention is help the team and stop us from having to manually calculate total recording time.

Project Layout
———————
recordingTime.go - The main project that grabs recording data and convert to total time

config.json - Contains graphQL token and HTTP time out configuration. This will have to be revised with your custom graphQL API token

recordingTime_test.go - Contains tests to verify that the recordingTime still operates correctly after changes are made to recordingTime.go. Run with “go test”.



