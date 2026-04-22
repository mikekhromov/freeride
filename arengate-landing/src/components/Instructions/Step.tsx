import { useIntersectionObserver } from '../../hooks/useIntersectionObserver'
import styles from './Instructions.module.css'

interface StepProps {
  number: string
  title: string
  description: string
  index: number
}

export default function Step({ number, title, description, index }: StepProps) {
  const { ref, isVisible } = useIntersectionObserver<HTMLDivElement>({
    threshold: 0.25,
    once: true,
  })

  return (
    <div
      ref={ref}
      className={`${styles.step} ${isVisible ? styles.visible : ''}`}
      style={{ transitionDelay: `${index * 150}ms` }}
    >
      <div className={styles.number}>{number}</div>
      <div className={styles.textWrap}>
        <h3 className={styles.stepTitle}>{title}</h3>
        <p className={styles.stepDescription}>{description}</p>
      </div>
    </div>
  )
}
