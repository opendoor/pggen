package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/opendoor-labs/pggen/examples/include_specs/models"
	"github.com/opendoor-labs/pggen/include"
)

func main() {
	ctx := context.Background()

	conn, err := sql.Open("postgres", "postgres://localhost/pggen_example?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	pgClient := models.NewPGClient(conn)

	//
	// add our linear family to the database
	//

	sueID, err := pgClient.InsertGrandparent(ctx, &models.Grandparent{
		Name: "Sue Slygh",
	})
	if err != nil {
		log.Fatal(err)
	}
	paulID, err := pgClient.InsertParent(ctx, &models.Parent{
		Name: "Paul Slygh",
		GrandparentId: sueID,
	})
	if err != nil {
		log.Fatal(err)
	}
	alexisID, err := pgClient.InsertChild(ctx, &models.Child{
		Name: "Alexis Slygh",
		ParentId: paulID,
	})
	if err != nil {
		log.Fatal(err)
	}
	// update Sue to make Alexis her favorite
	sue, err := pgClient.GetGrandparent(ctx, sueID)
	if err != nil {
		log.Fatal(err)
	}
	sue.FavoriteGrandkidId = &alexisID
	_, err = pgClient.UpdateGrandparent(ctx, sue, models.GrandparentAllFields)
	if err != nil {
		log.Fatal(err)
	}

	//
	// Now use include specs to fill in the `sue` object
	//

	// a basic include spec
	fmt.Println("spec: grandparents.parents")
	spec := include.Must(include.Parse("grandparents.parents"))
	err = pgClient.GrandparentFillIncludes(ctx, sue, spec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Parent is:", sue.Parents[0].Name)
	fmt.Println("Childen of parent is:", sue.Parents[0].Children)

	// go all the way to children
	sue, err = pgClient.GetGrandparent(ctx, sueID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nspec: grandparents.parents.children")
	spec = include.Must(include.Parse("grandparents.parents.children"))
	err = pgClient.GrandparentFillIncludes(ctx, sue, spec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Parent is:", sue.Parents[0].Name)
	fmt.Println("Child of parent is:", sue.Parents[0].Children[0].Name)
	fmt.Println("Favorite Grandkid is:", sue.FavoriteGrandkid)

	// fill in the pointer that goes back from the parent struct to the grandparent struct
	sue, err = pgClient.GetGrandparent(ctx, sueID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nspec: grandparents.parents.{grandparents,children}")
	spec = include.Must(include.Parse("grandparents.parents.{grandparents,children}"))
	err = pgClient.GrandparentFillIncludes(ctx, sue, spec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Parent is:", sue.Parents[0].Name)
	fmt.Println("Child of parent is:", sue.Parents[0].Children[0].Name)
	fmt.Println("Parent of parent is:", sue.Parents[0].Grandparent.Name)
	fmt.Println("Favorite Grandkid is:", sue.FavoriteGrandkid)

	// fill in the favorite grandkid reference
	// Note the way that we have to tell it which table the custom name is refering to.
	sue, err = pgClient.GetGrandparent(ctx, sueID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nspec: grandparents.favorite_grandkid->children")
	spec = include.Must(include.Parse("grandparents.favorite_grandkid->children"))
	err = pgClient.GrandparentFillIncludes(ctx, sue, spec)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Parents is:", sue.Parents)
	fmt.Println("FavoriteGrandkid is:", sue.FavoriteGrandkid.Name)

	// Use the pggen-generated include spec to fill everything.
	// Be careful with this.
	sue, err = pgClient.GetGrandparent(ctx, sueID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nspec: models.GrandparentAllIncludes")
	err = pgClient.GrandparentFillIncludes(ctx, sue, models.GrandparentAllIncludes)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Child of parent is:", sue.Parents[0].Children[0].Name)
	fmt.Println("Parent of parent is:", sue.Parents[0].Grandparent.Name)
	fmt.Println("FavoriteGrandkid is:", sue.FavoriteGrandkid.Name)
}
