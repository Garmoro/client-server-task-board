import { IsIn, IsString, MaxLength, MinLength } from 'class-validator';

export class CreateMessageDto {
  @IsIn(['user', 'operator', 'bot', 'system'])
  sender!: string;

  @IsString()
  @MinLength(1)
  @MaxLength(20000)
  content!: string;
}
