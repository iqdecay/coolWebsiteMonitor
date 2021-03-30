# WEBSITE MONITOR WITH CLI INTERFACE

## Requirements :

- go 1.13

The modules are handled via `go modules`, there is no need to install it

## Usage :

### Running the program :
- Build the program for your machine : `go build -race coolMonitor`. It will install all 
the necessary modules

 The `-race` flag is there to ensure that any data race is detected, but normally
there aren't any
- Create an input file following the example of `websites.txt`
- Run the binary : `./coolMonitor -f input_filename`

If you don't specify the flag `-f`, the default file `websites.txt` will be used

### Navigating the UI :

- Use the left and right arrows to switch from one pane to another
- Hit 'Enter' to enable scrolling, and scroll using the up and  down arrow keys

### Running tests : 

Only unit tests for the alerting logic are implemented.

To run them : `go test -race -v`

Normally, all tests pass

## Design choices with this application 

- Using Golang seemed like an obvious choice. It has a very simple yet very
  effective system for handling concurrency.

- I chose to use a third-party module,
  [gocui](https://github.com/jroimartin/gocui), to display a nice-looking
  interface for the logs and alert. The repository has around 7k stars on
  Github, and the code is easily readable, plus I had prior experience with it,
  so it was a natural choice.


## Improvements :

- The HTTP status code statistics are stored but actually never used. One improvement could be to
display the top 3 most common status codes for a given website, with their incidence.

- I feel like the code isn't general enough, for instance when creating monitors :
the WebsiteMonitor type could have a collection of WebsiteStatistics, so that the intervals 
can be changed at will, and not be pre-determined

- The alerting logic tests are currently working, but the "Quick" one actually doesn't work if you
add some delay in the HTTP response. I was not able to understand why, it probably has something to
do with the way availability is calculated : the delay of the HTTP response is in generall order of
magnitude longer than the monitoring interval of 10ms, so there might be some issues there

- Improvements could be made with the Mutex, they are locked during whole function calls, maybe
their locking could be more atomic.

- There is a useless channel passed as arguments to the monitors. It is there so that when runnning
tests, monitoring can be shut-down so as to avoid errors when the monitors are unable to connect The
improvement to be made would be to collect system signals so that the monitoring routines can be
shut-down properly, even outside tests.

- The TTFB is currently dependent on the DNS lookup time (which is not per-say part of the website
performance), so improvements could be made to not take it into account.

- The user should have the possibility to put the logs in a file, in addition 
to seeing them in the UI.

- It would be useful to be able to customize, using the UI, the wesites checked (add or remove ones
or change the interval at which they are checked.


