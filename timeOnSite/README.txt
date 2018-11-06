Hello!

This project is used to recreate the Samsara Time on Site report using graphQL. Its intention is to teach me more about concurrency and golang in general. 

Project Layout
———————
timeOnSite.go - The main project that executes the time on site report
Can be run with:
“./timeOnSite <groupID> <endTimeMs> <durationMs> <itemize trips (bool)>" from command line

config.json - Contains graphQL token and other configs. You will have to enter your own API token for script to run

timeOnSite_test.go - Tests the functions of timeOnSite to verify if there are any breaking changes from main

