// +build appengine
// NeuralGo AppEngine interface. Supports network creation / loading, training,
// testing, evaluation, and serialization.

package appengine

import (
  "appengine";
  "appengine/memcache";
  "encoding/json";
  "fmt";
  "github.com/golang/protobuf/proto";
  "math/rand";
  "net/http";
  "strconv";
  "time";
  "neural"
)

func init() {
  http.HandleFunc("/create", create)
  http.HandleFunc("/train", train)
  http.HandleFunc("/test", test)
  http.HandleFunc("/evaluate", evaluate)
  http.HandleFunc("/get", get)
}

func unmarshal(data []byte, v interface{}, c appengine.Context,
               w http.ResponseWriter) bool {
  err := json.Unmarshal(data, v)
  if err != nil {
    c.Errorf("Could not unmarshal data with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return false
  }
  return true
}

func getModelFromCache(
    modelId string, c appengine.Context, w http.ResponseWriter) (
    neuralNetwork neural.Network, success bool) {
  var byteNetwork *memcache.Item
  var err error
  if byteNetwork, err = memcache.Get(c, modelId); err != nil {
    c.Errorf("Could not retrieve neural network with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    success = false
    return
  }

  if err  = neuralNetwork.Deserialize(byteNetwork.Value); err != nil {
    c.Errorf("Could not deserialize neural network with error: %s", err.Error())
    success = false
    return
  }
  success = true
  return
}

// neuralNetwork will be placed into memcache with key modelId, unless modelId
// is empty, in which case the current time will be used.
func putModelIntoCache(
    modelId string, neuralNetwork neural.Network, c appengine.Context,
    w http.ResponseWriter) (newModelId string, success bool) {
  // Copy modelId into return if it was provided.
  if len(modelId) == 0 {
    newModelId = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
  } else {
    newModelId = modelId
  }
  item := &memcache.Item{
    Key: newModelId,
    Value: neuralNetwork.Serialize(),
  }
  if err := memcache.Set(c, item); err != nil {
    c.Errorf("Could not store neural network with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    success = false
    return
  }
  success = true
  return
}

func create(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  // Get params from request.
  err := r.ParseForm()
  if err != nil {
    c.Errorf("Could not parse form with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  rand.Seed(time.Now().UTC().UnixNano())

  neuralNetwork := new(neural.Network)
  if err  = neuralNetwork.Deserialize([]byte(r.FormValue("serializedNetwork")));
     err != nil {
    c.Errorf("Could not deserialize neural network with error: %s", err.Error())
    return
  }
  // If synapse weights aren't specified, randomize them.
  if neuralNetwork.Layers[0].Weight.At(0, 0) == 0 {
    neuralNetwork.RandomizeSynapses()
  }
  var modelId string
  var success bool
  if modelId, success = putModelIntoCache("", *neuralNetwork, c, w); !success {
    return
  }
  w.Write([]byte(modelId))
}

func train(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  // Get params from request.
  err := r.ParseForm()
  if err != nil {
    c.Errorf("Could not parse form with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  var neuralNetwork neural.Network
  var success bool
  if neuralNetwork, success = getModelFromCache(r.FormValue("modelId"), c, w);
     !success {
    return
  }
  trainingExamples := make([]neural.Datapoint, 0)
  if !unmarshal([]byte(r.FormValue("trainingExamples")), &trainingExamples, c,
                w) {
    return
  }
  var trainingIterations int
  trainingIterations, err = strconv.Atoi(r.FormValue("trainingIterations"))
  if err != nil {
    c.Errorf("Could not parse trainingIterations with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var learningRate float64
  learningRate, err = strconv.ParseFloat(r.FormValue("learningRate"), 64)
  if err != nil {
    c.Errorf("Could not parse learningRate with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var weightDecay float64
  weightDecay, err = strconv.ParseFloat(r.FormValue("weightDecay"), 64)
  if err != nil {
    c.Errorf("Could not parse weightDecay with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var batchSize int
  batchSize, err = strconv.Atoi(r.FormValue("batchSize"))
  if err != nil {
    c.Errorf("Could not parse batchSize with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var errorName neural.ErrorName
  errorName, err = neural.ErrorName(strconv.Atoi(r.FormValue("errorName")))
  if err != nil {
    c.Errorf("Could not parse errorName with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Train the model.
  learningConfiguration := neural.LearningConfiguration{
      Epochs: proto.Int32(int32(trainingIterations)),
      Rate: proto.Float64(learningRate),
      Decay: proto.Float64(weightDecay),
      BatchSize: proto.Int32(int32(batchSize)),
      ErrorName: errorName.Enum(),
  }
  neural.Train(&neuralNetwork, trainingExamples, learningConfiguration)
  if _, success := putModelIntoCache(
         r.FormValue("modelId"), neuralNetwork, c, w); !success {
    return
  }
  w.Write([]byte(fmt.Sprintf(
      "Training error: %v\n",
      neural.Evaluate(neuralNetwork, trainingExamples))))
}

func test(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  // Get params from request.
  err := r.ParseForm()
  if err != nil {
    c.Errorf("Could not parse form with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  var neuralNetwork neural.Network
  var success bool
  if neuralNetwork, success = getModelFromCache(r.FormValue("modelId"), c, w);
     !success {
    return
  }
  testingExamples := make([]neural.Datapoint, 0)
  if !unmarshal([]byte(r.FormValue("testingExamples")), &testingExamples, c,
                w) {
    return
  }

  // Test the model.
  w.Write([]byte(fmt.Sprintf(
    "Testing error: %v\n", neural.Evaluate(neuralNetwork, testingExamples))))
}

func evaluate(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  // Get params from request.
  err := r.ParseForm()
  if err != nil {
    c.Errorf("Could not parse form with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  var neuralNetwork neural.Network
  var success bool
  if neuralNetwork, success = getModelFromCache(r.FormValue("modelId"), c, w);
     !success {
    return
  }
  features := make([]float64, 0)
  if !unmarshal([]byte(r.FormValue("features")), &features, c, w) {
    return
  }

  // Evaluate the example.
  w.Write([]byte(fmt.Sprintf(
    "Evaluation: %v\n", neuralNetwork.Evaluate(features))))
}

func get(w http.ResponseWriter, r *http.Request) {
  c := appengine.NewContext(r)
  // Get params from request.
  err := r.ParseForm()
  if err != nil {
    c.Errorf("Could not parse form with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Get the model.
  // TODO(ariw): Switch this to getModelFromCache?
  var byteNetwork *memcache.Item
  if byteNetwork, err = memcache.Get(c, r.FormValue("modelId")); err != nil {
    c.Errorf("Could not retrieve neural network with error: %s", err.Error())
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  w.Write([]byte(fmt.Sprintf("Network: %v\n", string(byteNetwork.Value))))
}
