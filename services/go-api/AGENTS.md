##Architechture
-This folder contains the Authentication and Go API service
-The server runs on port 8080
-The database in use is PostreSQL, managed via GORM (gorm.io/gorm and gorm.io/driver/postgres).
-Database models and AutoMigrate live in services/go-api/models and services/go-api/database.
-The prisma scehma is managed at /prisma/schema.prisma

##Code quality
-Database access is fully handled via this Go API.
-A function should always do exactly what is explains
-Function should do one job
-Reusablity of functions are very important
-Function name should describe what the function about
-Do not use shortend function names, name should be descriptive even though they become long

##Restrictions
-Do not install new dependencies without approval
-Do not modify environment variables
-Never expose secrets or environment variables to client-side components
-Do not delete existing function when implementing new once, untill proper tests are performed
-Do not run destructive commands without approval

##Tests
-Every feature should should include a go unit test
