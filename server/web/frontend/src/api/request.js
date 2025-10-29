import axios from 'axios'

// 创建axios实例
const request = axios.create({
  baseURL: '', // 使用相对路径，通过vite代理
  timeout: 10000, // 请求超时时间
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
request.interceptors.request.use(
  config => {
    // 在发送请求之前做些什么
    console.log('Request:', config.method?.toUpperCase(), config.url)
    
    // 可以在这里添加token等认证信息
    // const token = localStorage.getItem('token')
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`
    // }
    
    return config
  },
  error => {
    // 对请求错误做些什么
    console.error('Request Error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  response => {
    // 对响应数据做点什么
    console.log('Response:', response.status, response.config.url)
    
    // 统一处理响应数据
    const { data } = response
    
    // 如果后端返回的数据格式是 { success: true, data: ... }
    if (data && typeof data.success !== 'undefined') {
      if (!data.success) {
        // 业务错误
        const error = new Error(data.error_message || data.message || 'Business Error')
        error.code = data.code
        return Promise.reject(error)
      }
      // 返回实际数据
      return data.data || data
    }
    
    // 直接返回数据
    return data
  },
  error => {
    // 对响应错误做点什么
    console.error('Response Error:', error)
    
    let message = 'Network Error'
    
    if (error.response) {
      // 服务器返回了错误状态码
      const { status, data } = error.response
      
      switch (status) {
        case 400:
          message = data?.message || 'Bad Request'
          break
        case 401:
          message = 'Unauthorized'
          // 可以在这里处理登录过期
          // window.location.href = '/login'
          break
        case 403:
          message = 'Forbidden'
          break
        case 404:
          message = 'Not Found'
          break
        case 500:
          message = 'Internal Server Error'
          break
        default:
          message = data?.message || `Error ${status}`
      }
    } else if (error.request) {
      // 请求已发出但没有收到响应
      message = 'No response from server'
    } else {
      // 其他错误
      message = error.message
    }
    
    // 创建统一的错误对象
    const customError = new Error(message)
    customError.status = error.response?.status
    customError.code = error.code
    
    return Promise.reject(customError)
  }
)

export default request