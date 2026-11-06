##Architechture
-This folder contains the Python Health Analytics & AI service
-The server runs on port 8000
-This service used Pandas, Numpy
-This service uses a 3-layer SciSpacy → Prompt → LLaMA pipeline for AI insights.
-LAYER 1: SciSpacy Pre-Processor (Local CPU) │ Tokenizes & scans biomedical vocabulary | Extracts: Conditions, Drugs, Dosages
-LAYER 2: Dynamic Prompt Constructor │ Merges structured data into clean prompt │ Applies LLaMA chat template
LAYER 3: Fine-Tuned LLaMA (LoRA Backend) │ Receives clean, pre-parsed parameters │ Generates insights + diet plan in JSON

##Code quality
-A function should always do exactly what is explains
-Function should do one job
-Reusablity of functions are very important
-Function name should describe what the function about
-Do not use shortend function names, name should be descriptive even though they become long

##Restrictions
-Do not access database directly without calling the server
-Do not install new dependencies without approval
-Do not modify environment variables
-Never expose secrets or environment variables to client-side components
-Do not delete existing function when implementing new once, untill proper tests are performed
-Do not run destructive commands without approval

##Tests
-Every feature should should include a pytest unit test
