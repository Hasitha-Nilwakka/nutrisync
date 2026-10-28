import LoginForm from "@/feature/auth/login/Login";
import RegistrationForm from "@/feature/auth/register/Register";

export default function Home() {
  return (
    <div>
      <RegistrationForm/>
      <LoginForm/>
    </div>
  );
}
