// +build !appengine
// NeuralGo command line interface. Supports network creation / loading,
// training, testing, and serialization. MNIST data is supported as a motivating
// example.
//
// Sample usage:
// go run cmdline.go -serialized_network_file network.txt -training_file training.txt -testing_file testing.txt

package main

import (
  "encoding/json";
  "flag";
  "fmt";
  "github.com/golang/protobuf/proto";
  "github.com/petar/GoMNIST";
  "io/ioutil";
  "log";
  "math/rand";
  "os";
  "runtime/pprof";
  "time";
  "./neural"
)

var serializedNetworkFlag = flag.String(
  "serialized_network", "", "File with JSON-formatted NetworkConfiguration.")
var mnistFlag = flag.String(
  "mnist", "",
  "Location of MNIST training / testing data. If non-empty, overrides " +
  "-training_file and -testing_file.")
var trainingExamplesFlag = flag.String(
  "training_file", "",
  "File with JSON-formatted array of training examples with values.")
var testingExamplesFlag = flag.String(
  "testing_file", "",
  "File with JSON-formatted array of testing examples with values.")
var trainingIterationsFlag = flag.Int(
  "training_iterations", 1000, "Number of training iterations.")
var learningRateFlag = flag.Float64(
  "learning_rate", 0.001, "Speed of training.")
var weightDecayFlag = flag.Float64(
  "weight_decay", 0, "Weight decay rate.")
var batchSizeFlag = flag.Int(
  "batch_size", 1, "Size of batches used for training.")
var errorNameFlag = flag.String(
  "error_name", "QUADRATIC_COST", "Which error function to use for training.")
var serializedNetworkOutFlag = flag.String(
  "serialized_network_out", "",
  "File to write JSON-formatted NetworkConfiguration.")
var cpuProfileFlag = flag.String(
  "cpu_profile", "", "Write CPU profile to file.")

func ReadDatapointsOrDie(filename string) []neural.Datapoint {
  bytes, err := ioutil.ReadFile(filename)
  if err != nil {
    log.Fatal(err)
  }
  datapoints := make([]neural.Datapoint, 0)
  err = json.Unmarshal(bytes, &datapoints)
  if err != nil {
    log.Fatal(err)
  }
  return datapoints
}

func main() {
  flag.Parse()
  if *cpuProfileFlag != "" {
    f, err := os.Create(*cpuProfileFlag)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
  }

  rand.Seed(time.Now().UTC().UnixNano())

  // Set up neural network.
  var neuralNetwork *neural.Network
  var trainingExamples []neural.Datapoint
  var testingExamples []neural.Datapoint
  if len(*mnistFlag) > 0 {
    train, test, err := GoMNIST.Load(*mnistFlag)
    if err != nil {
      log.Fatal(err)
    }
    for i := 0; i < train.Count(); i++ {
      var datapoint neural.Datapoint
      image, label := train.Get(i)
      datapoint.Values = append(datapoint.Values, float64(label))
      for _, pixel := range(image) {
        datapoint.Features = append(datapoint.Features, float64(pixel))
      }
      trainingExamples = append(trainingExamples, datapoint)
    }
    for i := 0; i < test.Count(); i++ {
      var datapoint neural.Datapoint
      image, label := test.Get(i)
      datapoint.Values = append(datapoint.Values, float64(label))
      for _, pixel := range(image) {
        datapoint.Features = append(datapoint.Features, float64(pixel))
      }
      testingExamples = append(testingExamples, datapoint)
    }
  } else {
    trainingExamples = ReadDatapointsOrDie(*trainingExamplesFlag)
    testingExamples = ReadDatapointsOrDie(*testingExamplesFlag)
  }
  fmt.Printf("Finished loading data!\n")

  byteNetwork, err := ioutil.ReadFile(*serializedNetworkFlag)
  if err != nil {
    log.Fatal(err)
  }
  neuralNetwork = new(neural.Network)
  neuralNetwork.Deserialize(byteNetwork)
  // If synapse weights aren't specified, randomize them.
  if neuralNetwork.Layers[0].Weight.At(0, 0) == 0 {
    neuralNetwork.RandomizeSynapses()
  }
  fmt.Printf("Finished creating the network!\n")

  // Train the model.
  learningConfiguration := neural.LearningConfiguration{
      Epochs: proto.Int32(int32(*trainingIterationsFlag)),
      Rate: proto.Float64(*learningRateFlag),
      Decay: proto.Float64(*weightDecayFlag),
      BatchSize: proto.Int32(int32(*batchSizeFlag)),
      ErrorName:
          neural.ErrorName(neural.ErrorName_value[*errorNameFlag]).Enum(),
  }
  neural.Train(neuralNetwork, trainingExamples, learningConfiguration)

  // Test & output model:
  fmt.Printf("Training error: %v\nTesting error: %v\n",
             neural.Evaluate(*neuralNetwork, trainingExamples),
             neural.Evaluate(*neuralNetwork, testingExamples))
  if len(*serializedNetworkOutFlag) > 0 {
    ioutil.WriteFile(*serializedNetworkOutFlag, neuralNetwork.Serialize(), 0777)
  }
}
