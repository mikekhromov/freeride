import Step from './Step'
import styles from './Instructions.module.css'

const steps = [
  {
    number: '01',
    title: 'Напиши боту @arengate_bot',
    description: 'Нажми /start для начала',
  },
  {
    number: '02',
    title: 'Запроси доступ',
    description: 'Нажми кнопку "Запросить"',
  },
  {
    number: '03',
    title: 'Дождись одобрения',
    description: 'Администратор рассмотрит заявку в течение 24 часов',
  },
  {
    number: '04',
    title: 'Получи ссылки',
    description: 'VPN подписка + Telegram прокси придут в личку от бота',
  },
  {
    number: '05',
    title: 'Подключись',
    description: 'Установи Happ (iOS/Android) или любой VLESS клиент',
  },
]

export default function Instructions() {
  return (
    <section className={styles.section} aria-label="Как начать">
      <div className={styles.container}>
        <h2 className={styles.heading}>Как начать</h2>
        <div className={styles.list}>
          {steps.map((step, index) => (
            <Step
              key={step.number}
              number={step.number}
              title={step.title}
              description={step.description}
              index={index}
            />
          ))}
        </div>
      </div>
    </section>
  )
}
