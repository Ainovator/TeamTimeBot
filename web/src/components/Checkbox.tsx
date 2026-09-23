import { forwardRef, type InputHTMLAttributes } from 'react'
import './checkbox.css'

export type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>

/** Native checkbox semantics and form events, with the shared TeamTime appearance. */
export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(function Checkbox({ className = '', ...props }, ref) {
  return <input {...props} ref={ref} type="checkbox" className={`tt-checkbox ${className}`.trim()}/>
})
