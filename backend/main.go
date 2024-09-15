package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Task struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title,omitempty" bson:"title,omitempty"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	DueDate     string             `json:"dueDate,omitempty" bson:"dueDate,omitempty"`
	Status      string             `json:"status,omitempty" bson:"status,omitempty"`
}

var client *mongo.Client

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	router := mux.NewRouter()

	// Routes for the task tracker
	router.HandleFunc("/tasks", createTask).Methods("POST")
	router.HandleFunc("/tasks", getTasks).Methods("GET")
	router.HandleFunc("/tasks/{id}", getTaskByID).Methods("GET")
	router.HandleFunc("/tasks/{id}", updateTask).Methods("PUT")
	router.HandleFunc("/tasks/{id}", deleteTask).Methods("DELETE")

	// CORS configuration
	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:3000"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type"})

	port := os.Getenv("PORT")
	fmt.Printf("Server running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(router)))
}

func getCollection() *mongo.Collection {
	return client.Database(os.Getenv("MONGO_DB")).Collection("tasks")
}

// CRUD Operations for Tasks

// Create a new task
func createTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var task Task
	_ = json.NewDecoder(r.Body).Decode(&task)
	task.ID = primitive.NewObjectID()

	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println(task)
	_, err := collection.InsertOne(ctx, task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Get all tasks
func getTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var tasks []Task
	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var task Task
		cursor.Decode(&task)
		tasks = append(tasks, task)
	}

	json.NewEncoder(w).Encode(tasks)
}

// Get a specific task by ID
func getTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	id, _ := primitive.ObjectIDFromHex(params["id"])
	var task Task
	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Update an existing task
func updateTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	id, _ := primitive.ObjectIDFromHex(params["id"])
	var updatedTask Task
	_ = json.NewDecoder(r.Body).Decode(&updatedTask)

	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": updatedTask}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(updatedTask)
}

// Delete a task
func deleteTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	id, _ := primitive.ObjectIDFromHex(params["id"])
	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode("Task deleted")
}
