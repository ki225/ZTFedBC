package main

import (
  "encoding/json"
  "fmt"
  "log"
  "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract defines the federated learning contract
type SmartContract struct {
  contractapi.Contract
}

// Model represents a federated learning model
type Model struct {
  ModelID      string `json:"modelID"`
  Owner        string `json:"owner"`
  ModelHash    string `json:"modelHash"`
  Aggregated   bool   `json:"aggregated"`
  Contributors []string `json:"contributors"`
}

// InitLedger initializes the ledger with a base model
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
  models := []Model{
    {ModelID: "model1", Owner: "admin", ModelHash: "abc123", Aggregated: false, Contributors: []string{}},
  }

  for _, model := range models {
    modelJSON, err := json.Marshal(model)
    if err != nil {
      return err
    }

    err = ctx.GetStub().PutState(model.ModelID, modelJSON)
    if err != nil {
      return fmt.Errorf("failed to put model in ledger: %v", err)
    }
  }

  return nil
}

// SubmitModelUpdate allows nodes to submit local model updates
func (s *SmartContract) SubmitModelUpdate(ctx contractapi.TransactionContextInterface, modelID string, modelHash string) error {
  modelJSON, err := ctx.GetStub().GetState(modelID)
  if err != nil {
    return fmt.Errorf("failed to get model: %v", err)
  }
  if modelJSON == nil {
    return fmt.Errorf("model %s does not exist", modelID)
  }

  var model Model
  err = json.Unmarshal(modelJSON, &model)
  if err != nil {
    return err
  }

  // Get submitter identity
  clientID, err := ctx.GetClientIdentity().GetID()
  if err != nil {
    return fmt.Errorf("failed to get client identity: %v", err)
  }

  // Zero Trust: Verify client authorization
  if !s.isAuthorized(clientID) {
    return fmt.Errorf("access denied: unauthorized client")
  }

  // Store new update
  model.ModelHash = modelHash
  model.Contributors = append(model.Contributors, clientID)

  updatedModelJSON, err := json.Marshal(model)
  if err != nil {
    return err
  }

  return ctx.GetStub().PutState(modelID, updatedModelJSON)
}

// AggregateModel finalizes the model after validation
func (s *SmartContract) AggregateModel(ctx contractapi.TransactionContextInterface, modelID string) error {
  modelJSON, err := ctx.GetStub().GetState(modelID)
  if err != nil {
    return fmt.Errorf("failed to get model: %v", err)
  }
  if modelJSON == nil {
    return fmt.Errorf("model %s does not exist", modelID)
  }

  var model Model
  err = json.Unmarshal(modelJSON, &model)
  if err != nil {
    return err
  }

  if len(model.Contributors) < 2 { // Example consensus: at least 2 updates
    return fmt.Errorf("not enough contributors for aggregation")
  }

  model.Aggregated = true
  updatedModelJSON, err := json.Marshal(model)
  if err != nil {
    return err
  }

  return ctx.GetStub().PutState(modelID, updatedModelJSON)
}

// isAuthorized enforces zero-trust access control (simplified)
func (s *SmartContract) isAuthorized(clientID string) bool {
  allowedClients := []string{"Org1MSP", "Org2MSP"} // Replace with actual verification
  for _, id := range allowedClients {
    if clientID == id {
      return true
    }
  }
  return false
}

// ReadModel retrieves the model details
func (s *SmartContract) ReadModel(ctx contractapi.TransactionContextInterface, modelID string) (*Model, error) {
  modelJSON, err := ctx.GetStub().GetState(modelID)
  if err != nil {
    return nil, fmt.Errorf("failed to read model: %v", err)
  }
  if modelJSON == nil {
    return nil, fmt.Errorf("model %s does not exist", modelID)
  }

  var model Model
  err = json.Unmarshal(modelJSON, &model)
  if err != nil {
    return nil, err
  }

  return &model, nil
}

// GetAllModels retrieves all models
func (s *SmartContract) GetAllModels(ctx contractapi.TransactionContextInterface) ([]*Model, error) {
  resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
  if err != nil {
    return nil, err
  }
  defer resultsIterator.Close()

  var models []*Model
  for resultsIterator.HasNext() {
    queryResponse, err := resultsIterator.Next()
    if err != nil {
      return nil, err
    }

    var model Model
    err = json.Unmarshal(queryResponse.Value, &model)
    if err != nil {
      return nil, err
    }
    models = append(models, &model)
  }

  return models, nil
}

func main() {
  modelChaincode, err := contractapi.NewChaincode(&SmartContract{})
  if err != nil {
    log.Panicf("Error creating federated learning chaincode: %v", err)
  }

  if err := modelChaincode.Start(); err != nil {
    log.Panicf("Error starting federated learning chaincode: %v", err)
  }
}
