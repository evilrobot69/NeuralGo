package neural

import (
  "github.com/gonum/matrix/mat64";
  "math/rand"
)

type Datapoint struct {
  Features []float64
  Values []float64
}

func min(a, b int) int {
  if a < b {
    return a
  }
  return b
}

func Train(neuralNetwork *Network, datapoints []Datapoint,
           learningConfiguration LearningConfiguration) {
  // Train on some number of iterations of permuted versions of the input.
  batchSize := int(*learningConfiguration.BatchSize)
  // Batch size 0 means do full batch learning.
  if batchSize == 0 {
    batchSize = len(datapoints)
  }
  error_function := NewErrorFunction(*learningConfiguration.ErrorName)
  features := mat64.NewDense(batchSize, len(datapoints[0].Features), nil)
  values := mat64.NewDense(batchSize, len(datapoints[0].Values), nil)
  for i := 0; i < int(*learningConfiguration.Epochs); i++ {
    perm := rand.Perm(len(datapoints))
    // TODO(ariw): This misses the last len(perm) % batchSize examples. Is this
    // okay?
    for j := 0; j <= len(perm) - batchSize; j += batchSize {
      for k := 0; k < batchSize; k++ {
        features.SetRow(k, datapoints[perm[j + k]].Features)
        values.SetRow(k, datapoints[perm[j + k]].Values)
      }
      neuralNetwork.Forward(features)
      neuralNetwork.Backward(values, error_function)
      neuralNetwork.Update(learningConfiguration)
    }
  }
}

// Return mean squared error of the network on datapoints.
func Evaluate(neuralNetwork Network, datapoints []Datapoint) float64 {
  square_error := 0.0
  for _, datapoint := range datapoints {
    output := neuralNetwork.Evaluate(datapoint.Features)
    for i, value := range datapoint.Values {
      square_error += (value - output[i]) * (value - output[i])
    }
  }
  return square_error / float64(len(datapoints))
}
