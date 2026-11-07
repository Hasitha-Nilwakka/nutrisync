import { toast } from "sonner";

export interface APIResponse {
    type: 'success' | 'info' | 'error',
    title: string,
    message: string,
}

interface redirectToastProps {
    type : 'success' | 'info' | 'error',
    title : string,
    message : string,
    redirectTo : string
}

export function showAPIResponseToast(response : APIResponse) {
    const {title, message, type} = response
    if (type === 'success') {
        toast.success(title, { position: "top-center", description : message, richColors : true})
    } else if (type === 'info') {
        toast.info(title, { position: "top-center", description : message, richColors : true})
    } else {
        toast.error(title, { position: "top-center", description : message, richColors : true})
    }
}

export function showRedirectToast(props : redirectToastProps) {
    const {type, title, message, redirectTo} = props

    const completeMsg = `${message}, redirecting to ${redirectTo}`

    if (type === 'success') {
        toast.success(title, { position: "top-center", description : completeMsg, richColors : true})
    } else if (type === 'info') {
        toast.info(title, { position: "top-center", description : completeMsg, richColors : true})
    } else {
        toast.error(title, { position: "top-center", description : completeMsg, richColors : true})
    }
}
