package main

/*
Internal Notes
Int value decoding
1: Recording
2: Not Recording Error
3: Not Recording Stopped
4: Camera Starting
5: Camera On

Will just need to find IntValue 1's then look to next status change, take difference. Aggregate the time

What we'll input:
deviceID
startTimeMs
endTimeMs (accept some input to mean live, perhaps 0)

What we'll output:

Segment list
-----
Start time
End time
Segment recording time
&
Total recording time

*/

func main() {

}
