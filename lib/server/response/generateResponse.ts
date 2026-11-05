import { Icon } from "next/dist/lib/metadata/types/metadata-types"

export interface APIResponse<T = unknown> {
    type : 'success' | 'error' | 'info',
    icon : Icon,
    status : number,
    message : string,
    title : string,
    data?: T
}

export interface Options<T> {
  message?: string;
  status?: number;
  icon?: Icon;
  title?: string;
  data?: T;
}

export class generateResponse {
    static success<T = unknown>(options : Options<T> = {}) : APIResponse<T> {
        const {
            message = 'Operation successfull',
            status = 200,
            icon = '✅',
            title  = 'Completed',
            data
        } = options
        return {
            type : 'success',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            status : status,
            ...(data !== undefined && { data }),
        }
    }

    static info<T = unknown>(options : Options<T> = {}) : APIResponse<T> {
        const {
            message = 'Action required',
            status = 200,
            icon = 'ℹ️',
            title  = 'Info',
            data
        } = options
        return {
            type : 'info',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            status : status,
            ...(data !== undefined && { data }),
        }
    }

    static networkError<T = unknown>(options : Options<T> = {}) : APIResponse<T> {
        const {
            message = 'Network error. Please try again later.',
            status = 503,
            icon = '⚠️',
            title  = 'Network error',
            data
        } = options
        return {
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            status : status,
            ...(data !== undefined && { data }),
        }
    }

    static internalServerError<T = unknown>(options : Options<T> = {}) : APIResponse<T> {
        const {
            message = 'Server error. Please try again later.',
            status = 500,
            icon = '🛜',
            title  = 'Internal error',
            data
        } = options
        return {
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            status : status,
            ...(data !== undefined && { data }),
        }
    }

    static clientError<T = unknown>(options : Options<T> = {}) : APIResponse<T> {
        const {
            message = 'An error occured',
            status = 400,
            icon = '🚫',
            title  = 'Error',
            data
        } = options
        return {
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            status : status,
            ...(data !== undefined && { data }),
        }
    }
}