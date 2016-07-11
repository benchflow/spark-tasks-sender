package main

import (
 	"fmt"
)

// Create new analyser worker
func NewAnalyserWorker(id int, workerQueue chan chan WorkRequest) AnalyserWorker {
  worker := AnalyserWorker{
    ID:          id,
    Work:        make(chan WorkRequest),
    WorkerQueue: workerQueue,
    QuitChan:    make(chan bool)}
  
  return worker
}

// Analyser worker struct
type AnalyserWorker struct {
  ID          int
  Work        chan WorkRequest
  WorkerQueue chan chan WorkRequest
  QuitChan    chan bool
}

// Start analyser worker
func (w *AnalyserWorker) Start() {
    go func() {
      for {
      	// Add the worker to the worker queue
        w.WorkerQueue <- w.Work
        
        select {
    	// Get work from the worker's work queue
        case work := <-w.Work:
        	// Submit script to Spark
			success := submitScript(work.SparkArgs, work.Script)
			if success {
				meetRequirement(work.ScriptName, work.TrialID, work.ExperimentID, work.Level)
			}
			// Increment counter for trials completed for the given script
			counterId := getCounterID(work.ExperimentID, work.ScriptName, work.ContainerID, work.HostID, work.CollectorName)
			trialCount.SetIfAbsent(counterId, 0)
			i, ok := trialCount.Get(counterId)
			if ok {
				trialCount.Set(counterId, i.(int)+1)
				}
			// Check for script completions, and send the work request to the analyser consumer queue to signal a script is finished and
			// reqs need to be checked again to launch more scripts.
			if work.Level == "trial" {
				isTrialComplete(work.TrialID)
				AnalyserDispatchRequestsQueue <- work
			}
			if work.Level == "experiment" {
				isExperimentComplete(work.ExperimentID)
			}
          
        // Stop if signalled
        case <-w.QuitChan:
          fmt.Printf("worker%d stopping\n", w.ID)
          return
        }
      }
    }()
}

// To stop the worker
func (w *AnalyserWorker) Stop() {
  go func() {
    w.QuitChan <- true
  }()
}