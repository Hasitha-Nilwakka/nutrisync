import { Icon } from "next/dist/lib/metadata/types/metadata-types"
import { NextResponse } from "next/server";

export interface APIResponse<T = unknown> {
    type : 'success' | 'error' | 'info',
    icon : Icon,
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
    static success<T = unknown>(options : Options<T> = {}) : NextResponse<APIResponse> {
        const {
            message = 'Operation successfull',
            status = 200,
            icon = '✅',
            title  = 'Completed',
            data
        } = options
        return NextResponse.json({
            type : 'success',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            ...(data !== undefined && { data }),
        }, {status : status})
    }

    static info<T = unknown>(options : Options<T> = {}) : NextResponse<APIResponse>  {
        const {
            message = 'Action required',
            status = 200,
            icon = 'ℹ️',
            title  = 'Info',
            data
        } = options
        return NextResponse.json({
            type : 'info',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            ...(data !== undefined && { data }),
        }, {status : status})
    }

    static networkError<T = unknown>(options : Options<T> = {}) : NextResponse<APIResponse> {
        const {
            message = 'Network error. Please try again later.',
            status = 503,
            icon = '⚠️',
            title  = 'Network error',
            data
        } = options
        return NextResponse.json({
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            ...(data !== undefined && { data }),
        }, {status : status})
    }

    static internalServerError<T = unknown>(options : Options<T> = {}) : NextResponse<APIResponse> {
        const {
            message = 'Server error. Please try again later.',
            status = 500,
            icon = '🛜',
            title  = 'Internal error',
            data
        } = options
        return NextResponse.json({
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            ...(data !== undefined && { data }),
        }, {status : status})
    }

    static clientError<T = unknown>(options : Options<T> = {}) : NextResponse<APIResponse>  {
        const {
            message = 'An error occured',
            status = 400,
            icon = '🚫',
            title  = 'Error',
            data
        } = options
        return NextResponse.json({
            type : 'error',
            message : `${message} ${icon}`,
            icon : icon,
            title : title,
            ...(data !== undefined && { data }),
        }, {status : status})
    }
}