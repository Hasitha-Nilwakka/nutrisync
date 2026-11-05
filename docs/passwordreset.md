#Resetting password

##OP flow

requesting a verification link
1. Send password reset verification ui -> Done
2. call api/auth/passwordreset -> Done
3. route.ts POST() call from frontend with emailid -> Done
4. pass the email to passwordreset server file and handle the operation -> Done
5. send the response to route and forward to fronetend Done
6. show the outcome - this can be implemented later, for now show alert -> reroute to home page -> Done

actual verification
1. user click the email Done
2. call api/auth/resetpassword GET() Done
3. get the token from url Done
4. pass to passwordreset server file Done
5. validate -> success -> return the token along with the response DONE
6. route.ts forward the token and reroute to actual password change screen /new-password?token=token done
6. fail -> return relevant message Done

password change
1. the ui at /new-password get the token from url
2. validate the password and confirm password with zod
3. use RHF for form validation
4. on submit call api/auth/resetpassword POST and pass newPassword & token
5. record changes in db and and validate return to login screen

what will be left to do
Proper ui
alerts