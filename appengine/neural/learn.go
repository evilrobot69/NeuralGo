package neural

import (
  "github.com/gonum/matrix/mat64";
  "math/rand"
)

type Datapoint struct {
  Features []float64
  Values []float64
}

func Train(neuralNetwork *Network, datapoints []Datapoint,
           learningConfiguration LearningConfiguration) {
  // Train on some number of iterations of permuted versions of the input.
  for i := 0; i < int(*learningConfiguration.Epochs); i++ {
    perm := rand.Perm(len(datapoints))
    batchSize := batchSize
    // Batch size 0 means do full batch learning.
    if batchSize == 0 {
      batchSize = len(datapoints)
    }
    features := mat64.NewDense(batchSize, len(neuralNetwork.Inputs), nil)
    values := mat64.NewDense(
        batchSize, len(neuralNetwork.Layers[len(neuralNetwork.Layers) - 1]),
        nil)
    for j := 0; j < len(perm); j += batchSize {
      for k := 0; k < batchSize && j + k < len(perm); k++ {
        features.SetRow(k, datapoints[perm[j + k].Features])
        values.SetRow(k, datapoints[perm[j + k].Values])
      }
      // This logic will train on len(datapoints) % batchSize datapoints twice.
      // Is this a problem?
      neuralNetwork.Forward(features)
      neuralNetwork.Backward(values)
      neuralNetwork.Update(learningConfiguration)
    }
  }
}

func Evaluate(neuralNetwork Network, datapoints []Datapoint) float64 {
  square_error := 0.0
  for _, datapoint := range datapoints {
    output := neuralNetwork.Evaluate(datapoint.Features)
    for i, value := range datapoint.Values {
      square_error += (value - output[i]) * (value - output[i]) / 2
    }
  }
  return square_error / float64(len(datapoints))
}
