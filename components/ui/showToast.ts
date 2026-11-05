import { APIResponse } from "@/lib/server/response/generateResponse";
import { toast } from "sonner";

export function showToast(response : APIResponse) {
    const {title, message, type} = response
    if (type === 'success') {
        toast.success(title, { position: "top-center", description : message, richColors : true})
    } else if (type === 'info') {
        toast.info(title, { position: "top-center", description : message, richColors : true})
    } else {
        toast.error(title, { position: "top-center", description : message, richColors : true})
    }
}