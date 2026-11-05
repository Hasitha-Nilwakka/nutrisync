how to handle numorous erros without retyping them everywhere directly returns a shadcn alert

mek sure to handle loading alerts as well

design
create a statci class generateResponse
.success

generate success response
generateSuccessResponse(message, status) {
    NextResponse {
        message : message,
        status : status
    }

    return next response
}